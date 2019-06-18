package cfn

import (
	"context"

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

var cfnClient *cloudformation.Client

func getClient() *cloudformation.Client {
	if cfnClient == nil {
		cfnClient = cloudformation.New(client.Config())
	}

	return cfnClient
}

func GetStackTemplate(stackName string) (string, client.Error) {
	req := getClient().GetTemplateRequest(&cloudformation.GetTemplateInput{
		StackName:     &stackName,
		TemplateStage: "Original", //"Processed"
	})

	res, err := req.Send(context.Background())
	if err != nil {
		return "", client.NewError(err)
	}

	return *res.TemplateBody, nil
}

func StackExists(stackName string) (bool, client.Error) {
	stacks, err := ListStacks()
	if err != nil {
		return false, err
	}

	for _, s := range stacks {
		if *s.StackName == stackName {
			return true, nil
		}
	}

	return false, nil
}

func ListStacks() ([]cloudformation.StackSummary, client.Error) {
	req := getClient().ListStacksRequest(&cloudformation.ListStacksInput{
		StackStatusFilter: liveStatuses,
	})

	stacks := make([]cloudformation.StackSummary, 0)

	p := cloudformation.NewListStacksPaginator(req)
	for p.Next(context.Background()) {
		stacks = append(stacks, p.CurrentPage().StackSummaries...)
	}

	return stacks, client.NewError(p.Err())
}

func DeleteStack(stackName string) client.Error {
	// Get the stack properties
	req := getClient().DeleteStackRequest(&cloudformation.DeleteStackInput{
		StackName: &stackName,
	})

	_, err := req.Send(context.Background())

	return client.NewError(err)
}

func GetStack(stackName string) (cloudformation.Stack, client.Error) {
	// Get the stack properties
	req := getClient().DescribeStacksRequest(&cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})

	res, err := req.Send(context.Background())
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

	res, err := req.Send(context.Background())
	if err != nil {
		return nil, client.NewError(err)
	}

	return res.StackResources, nil
}

func GetStackEvents(stackName string) ([]cloudformation.StackEvent, client.Error) {
	req := getClient().DescribeStackEventsRequest(&cloudformation.DescribeStackEventsInput{
		StackName: &stackName,
	})

	events := make([]cloudformation.StackEvent, 0)

	p := cloudformation.NewDescribeStackEventsPaginator(req)
	for p.Next(context.Background()) {
		events = append(events, p.CurrentPage().StackEvents...)
	}

	return events, client.NewError(p.Err())
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

	_, err := req.Send(context.Background())

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

	_, err := req.Send(context.Background())

	return client.NewError(err)
}

func Deploy(template string, params []cloudformation.Parameter, stackName string) client.Error {
	exists, err := StackExists(stackName)
	if err != nil {
		return err
	}

	if exists {
		return updateStack(template, params, stackName)
	}

	return createStack(template, params, stackName)
}

func WaitUntilStackExists(stackName string) client.Error {
	err := getClient().WaitUntilStackExists(context.Background(), &cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})

	return client.NewError(err)
}

func WaitUntilStackCreateComplete(stackName string) client.Error {
	err := getClient().WaitUntilStackCreateComplete(context.Background(), &cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})

	return client.NewError(err)
}
