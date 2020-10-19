package cfn

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws-cloudformation/rain/cfn"
	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/aws-cloudformation/rain/client"
	"github.com/aws/aws-sdk-go-v2/aws"
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

func getClient() *cloudformation.Client {
	return cloudformation.New(client.Config())
}

// GetStackTemplate returns the template used to launch the named stack
func GetStackTemplate(stackName string, processed bool) (string, client.Error) {
	templateStage := "Original"
	if processed {
		templateStage = "Processed"
	}

	req := getClient().GetTemplateRequest(&cloudformation.GetTemplateInput{
		StackName:     &stackName,
		TemplateStage: cloudformation.TemplateStage(templateStage),
	})

	res, err := req.Send(context.Background())
	if err != nil {
		return "", client.NewError(err)
	}

	return *res.TemplateBody, nil
}

// StackExists checks whether the named stack currently exists
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

// ListStacks returns a list of all existing stacks
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

// DeleteStack deletes a stack
func DeleteStack(stackName string) client.Error {
	// Get the stack properties
	req := getClient().DeleteStackRequest(&cloudformation.DeleteStackInput{
		StackName: &stackName,
	})

	_, err := req.Send(context.Background())

	return client.NewError(err)
}

// SetTerminationProtection enables or disables termination protection for a stack
func SetTerminationProtection(stackName string, protectionEnabled bool) client.Error {
	// Set termination protection
	req := getClient().UpdateTerminationProtectionRequest(&cloudformation.UpdateTerminationProtectionInput{
		StackName:                   &stackName,
		EnableTerminationProtection: aws.Bool(protectionEnabled),
	})

	_, err := req.Send(context.Background())

	if err != nil {
		return client.NewError(err)
	}

	return nil
}

// GetStack returns a cloudformation.Stack representing the named stack
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

// GetStackResources returns a list of the resources in the named stack
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

// GetStackEvents returns all events associated with the named stack
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

func makeTags(tags map[string]string) []cloudformation.Tag {
	out := make([]cloudformation.Tag, 0)

	for key, value := range tags {
		out = append(out, cloudformation.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	return out
}

// CreateChangeSet creates a changeset
func CreateChangeSet(template cfn.Template, params []cloudformation.Parameter, tags map[string]string, stackName string) (string, client.Error) {
	templateBody := format.Template(template, format.Options{})

	changeSetType := "CREATE"

	exists, err := StackExists(stackName)
	if err != nil {
		return "", err
	}

	if exists {
		changeSetType = "UPDATE"
	}

	changeSetName := stackName + "-" + fmt.Sprint(time.Now().Unix())

	req := getClient().CreateChangeSetRequest(&cloudformation.CreateChangeSetInput{
		ChangeSetType: cloudformation.ChangeSetType(changeSetType),
		ChangeSetName: &changeSetName,
		StackName:     &stackName,
		TemplateBody:  &templateBody,
		Tags:          makeTags(tags),
		Parameters:    params,
		Capabilities: []cloudformation.Capability{
			"CAPABILITY_NAMED_IAM",
			"CAPABILITY_AUTO_EXPAND",
		},
	})

	_, err = req.Send(context.Background())
	if err != nil {
		return changeSetName, err
	}

	err = getClient().WaitUntilChangeSetCreateComplete(context.Background(), &cloudformation.DescribeChangeSetInput{
		ChangeSetName: &changeSetName,
		StackName:     &stackName,
	})

	if err != nil {
		// Get reason for failure
		csr := getClient().DescribeChangeSetRequest(&cloudformation.DescribeChangeSetInput{
			ChangeSetName: &changeSetName,
			StackName:     &stackName,
		})

		info, err := csr.Send(context.Background())

		if err != nil {
			return changeSetName, err
		}

		return changeSetName, errors.New(*info.StatusReason)
	}

	return changeSetName, nil
}

// GetChangeSet returns the named changeset
func GetChangeSet(stackName, changeSetName string) (*cloudformation.DescribeChangeSetResponse, client.Error) {
	req := getClient().DescribeChangeSetRequest(&cloudformation.DescribeChangeSetInput{
		ChangeSetName: &changeSetName,
		StackName:     &stackName,
	})

	res, err := req.Send(context.Background())

	return res, client.NewError(err)
}

// ExecuteChangeSet executes the named changeset
func ExecuteChangeSet(stackName, changeSetName string) client.Error {
	req := getClient().ExecuteChangeSetRequest(&cloudformation.ExecuteChangeSetInput{
		ChangeSetName: &changeSetName,
		StackName:     &stackName,
	})

	_, err := req.Send(context.Background())

	return client.NewError(err)
}

// DeleteChangeSet deletes the named changeset
func DeleteChangeSet(stackName, changeSetName string) client.Error {
	req := getClient().DeleteChangeSetRequest(&cloudformation.DeleteChangeSetInput{
		ChangeSetName: &changeSetName,
		StackName:     &stackName,
	})

	_, err := req.Send(context.Background())

	return client.NewError(err)
}

// WaitUntilStackExists pauses execution until the named stack exists
func WaitUntilStackExists(stackName string) client.Error {
	err := getClient().WaitUntilStackExists(context.Background(), &cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})

	return client.NewError(err)
}

// WaitUntilStackCreateComplete pauses execution until the stack is completed (or fails)
func WaitUntilStackCreateComplete(stackName string) client.Error {
	err := getClient().WaitUntilStackCreateComplete(context.Background(), &cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})

	return client.NewError(err)
}
