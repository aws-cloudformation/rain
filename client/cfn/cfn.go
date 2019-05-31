package cfn

import (
	"github.com/aws-cloudformation/rain/client"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

var liveStatuses = []cloudformation.StackStatus{
	"CREATE_IN_PROGRESS",
	"CREATE_FAILED",
	"CREATE_COMPLETE",
	"ROLLBACK_IN_PROGRESS",
	"ROLLBACK_FAILED",
	"ROLLBACK_COMPLETE",
	"DELETE_IN_PROGRESS",
	"DELETE_FAILED",
	"UPDATE_IN_PROGRESS",
	"UPDATE_COMPLETE_CLEANUP_IN_PROGRESS",
	"UPDATE_COMPLETE",
	"UPDATE_ROLLBACK_IN_PROGRESS",
	"UPDATE_ROLLBACK_FAILED",
	"UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS",
	"UPDATE_ROLLBACK_COMPLETE",
	"REVIEW_IN_PROGRESS",
}

var cfnClient *cloudformation.CloudFormation

func getClient() *cloudformation.CloudFormation {
	if cfnClient == nil {
		cfnClient = cloudformation.New(client.GetConfig())
	}

	return cfnClient
}

func GetStackTemplate(stackName string) (string, client.Error) {
	req := getClient().GetTemplateRequest(&cloudformation.GetTemplateInput{
		StackName:     &stackName,
		TemplateStage: "Original", //"Processed"
	})

	res, err := req.Send()
	if err != nil {
		return "", client.NewError(err)
	}

	return *res.TemplateBody, nil
}

func StackExists(stackName string) (bool, client.Error) {
	req := getClient().ListStacksRequest(&cloudformation.ListStacksInput{
		StackStatusFilter: liveStatuses,
	})

	p := req.Paginate()

	for p.Next() {
		for _, s := range p.CurrentPage().StackSummaries {
			if *s.StackName == stackName {
				return true, nil
			}
		}
	}

	return false, client.NewError(p.Err())
}

func ListStacks(fn func(cloudformation.StackSummary)) client.Error {
	req := getClient().ListStacksRequest(&cloudformation.ListStacksInput{
		StackStatusFilter: liveStatuses,
	})

	p := req.Paginate()
	for p.Next() {
		for _, s := range p.CurrentPage().StackSummaries {
			fn(s)
		}
	}

	return client.NewError(p.Err())
}

func DeleteStack(stackName string) client.Error {
	// Get the stack properties
	req := getClient().DeleteStackRequest(&cloudformation.DeleteStackInput{
		StackName: &stackName,
	})

	_, err := req.Send()

	return client.NewError(err)
}

func GetStack(stackName string) (cloudformation.Stack, client.Error) {
	// Get the stack properties
	req := getClient().DescribeStacksRequest(&cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})

	res, err := req.Send()
	if err != nil {
		return cloudformation.Stack{}, client.NewError(err)
	}

	return res.Stacks[0], nil
}

func GetStackResources(stackName string) ([]cloudformation.StackResource, client.Error) {
	// Get the stack resources
	req := getClient().DescribeStackResourcesRequest(&cloudformation.DescribeStackResourcesInput{
		StackName: &stackName,
	})

	res, err := req.Send()
	if err != nil {
		return nil, client.NewError(err)
	}

	return res.StackResources, nil
}

func createStack(template string, params []cloudformation.Parameter, stackName string) client.Error {
	req := getClient().CreateStackRequest(&cloudformation.CreateStackInput{
		Capabilities: []cloudformation.Capability{
			"CAPABILITY_NAMED_IAM",
			"CAPABILITY_AUTO_EXPAND",
		},
		OnFailure:    "DELETE", // ROLLBACK or DELETE
		StackName:    &stackName,
		TemplateBody: &template,
		Parameters:   params,
	})

	_, err := req.Send()

	return client.NewError(err)
}

func updateStack(template string, params []cloudformation.Parameter, stackName string) client.Error {
	req := getClient().UpdateStackRequest(&cloudformation.UpdateStackInput{
		Capabilities: []cloudformation.Capability{
			"CAPABILITY_NAMED_IAM",
			"CAPABILITY_AUTO_EXPAND",
		},
		StackName:    &stackName,
		TemplateBody: &template,
		Parameters:   params,
	})

	_, err := req.Send()

	return client.NewError(err)
}

func Deploy(template string, params []cloudformation.Parameter, stackName string) client.Error {
	if stackExists(stackName) {
		return updateStack(template, params, stackName)
	}

	return createStack(template, params, stackName)
}

func WaitUntilStackExists(stackName string) client.Error {
	err := getClient().WaitUntilStackExists(&cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})

	return client.NewError(err)
}

func WaitUntilStackCreateComplete(stackName string) client.Error {
	err := getClient().WaitUntilStackCreateComplete(&cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})

	return client.NewError(err)
}

func stackExists(stackName string) bool {
	ch := make(chan bool)

	go func() {
		ListStacks(func(s cloudformation.StackSummary) {
			if *s.StackName == stackName {
				ch <- true
			}
		})

		// Default
		ch <- false
	}()

	return <-ch
}
