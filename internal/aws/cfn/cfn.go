//go:build !func_test

package cfn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	smithy "github.com/aws/smithy-go"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
)

var liveStatuses = []types.StackStatus{
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

func checkTemplate(template cft.Template) (string, error) {
	templateBody := format.String(template, format.Options{})

	if len(templateBody) > 460800 {
		return "", fmt.Errorf("template is too large to deploy")
	}

	if len(templateBody) > 51200 {
		config.Debugf("Template is too large to deploy directly; uploading to S3.")

		bucket := s3.RainBucket(false)

		key, err := s3.Upload(bucket, []byte(templateBody))

		return fmt.Sprintf("http://%s.s3.amazonaws.com/%s", bucket, key), err
	}

	return templateBody, nil
}

func getClient() *cloudformation.Client {
	return cloudformation.NewFromConfig(aws.Config())
}

// GetStackTemplate returns the template used to launch the named stack
func GetStackTemplate(stackName string, processed bool) (string, error) {
	templateStage := "Original"
	if processed {
		templateStage = "Processed"
	}

	res, err := getClient().GetTemplate(context.Background(), &cloudformation.GetTemplateInput{
		StackName:     &stackName,
		TemplateStage: types.TemplateStage(templateStage),
	})
	if err != nil {
		return "", err
	}

	return *res.TemplateBody, nil
}

// StackExists checks whether the named stack currently exists
func StackExists(stackName string) (bool, error) {
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
func ListStacks() ([]types.StackSummary, error) {
	stacks := make([]types.StackSummary, 0)

	var token *string

	for {
		res, err := getClient().ListStacks(context.Background(), &cloudformation.ListStacksInput{
			NextToken:         token,
			StackStatusFilter: liveStatuses,
		})

		if err != nil {
			return stacks, err
		}

		stacks = append(stacks, res.StackSummaries...)

		if res.NextToken == nil {
			break
		}

		token = res.NextToken
	}

	return stacks, nil
}

// ListStackSets returns a list of all existing stack sets
func ListStackSets() ([]types.StackSetSummary, error) {
	stackSets := make([]types.StackSetSummary, 0)

	var token *string

	for {
		res, err := getClient().ListStackSets(context.Background(), &cloudformation.ListStackSetsInput{
			NextToken: token,
		})

		if err != nil {
			return stackSets, err
		}

		stackSets = append(stackSets, res.Summaries...)

		if res.NextToken == nil {
			break
		}

		token = res.NextToken
	}

	return stackSets, nil
}

// ListStackSetInstances returns a list of all stack set instances for a given stack set
func ListStackSetInstances(stackSetName string) ([]types.StackInstanceSummary, error) {
	instances := make([]types.StackInstanceSummary, 0)
	var token *string

	for {
		res, err := getClient().ListStackInstances(context.Background(), &cloudformation.ListStackInstancesInput{
			NextToken:    token,
			StackSetName: &stackSetName,
		})

		if err != nil {
			return instances, err
		}

		instances = append(instances, res.Summaries...)

		if res.NextToken == nil {
			break
		}

		token = res.NextToken
	}

	return instances, nil
}

// ListLast10StackSetOperations returns a list of last 10 operations for a given stack sets
func ListLast10StackSetOperations(stackSetName string) ([]types.StackSetOperationSummary, error) {
	stackOps := make([]types.StackSetOperationSummary, 0)

	res, err := getClient().ListStackSetOperations(context.Background(), &cloudformation.ListStackSetOperationsInput{
		MaxResults:   ptr.Int32(10),
		StackSetName: &stackSetName,
	})

	if err != nil {
		return stackOps, err
	}

	stackOps = append(stackOps, res.Summaries...)
	return stackOps, nil
}

// GetStackSetOperationsResult  returns an operation result for a given stack sets operation id
func GetStackSetOperationsResult(stackSetName *string, operationId *string) (*types.StackSetOperationResultSummary, error) {
	res, err := getClient().ListStackSetOperationResults(context.Background(), &cloudformation.ListStackSetOperationResultsInput{
		MaxResults:   ptr.Int32(1),
		OperationId:  operationId,
		StackSetName: stackSetName,
	})

	if err == nil && res != nil && len(res.Summaries) == 1 {
		return &res.Summaries[0], err
	}
	return nil, nil
}

// DeleteStack deletes a stack
func DeleteStack(stackName string) error {
	// Get the stack properties
	_, err := getClient().DeleteStack(context.Background(), &cloudformation.DeleteStackInput{
		StackName: &stackName,
	})

	return err
}

// DeleteStackSet deletes a stack set
func DeleteStackSet(stackSetName string) error {
	_, err := getClient().DeleteStackSet(context.Background(), &cloudformation.DeleteStackSetInput{
		StackSetName: &stackSetName,
	})

	return err
}

// DeleteAllStackSetInstances deletes all instances for a given stack set
func DeleteAllStackSetInstances(stackSetName string, wait bool) error {
	stackSet, err := GetStackSet(stackSetName)
	if err != nil {
		fmt.Printf("Could not find stack set '%s'", stackSetName)
		return err
	}
	instances, err := ListStackSetInstances(*stackSet.StackSetName)
	if err != nil {
		fmt.Printf("Could not fetch instances for stack set '%s'", stackSetName)
		return err
	}
	var input = &cloudformation.DeleteStackInstancesInput{
		Accounts:     []string{},
		Regions:      []string{},
		RetainStacks: false, //TODO: add flag
		StackSetName: &stackSetName,
	}
	fmt.Println("\nStack set instances to be deleted:  ")
	for _, i := range instances {
		if i.StackInstanceStatus.DetailedStatus != types.StackInstanceDetailedStatusRunning { //TODO: do we need to wait only for RUNNING?
			fmt.Printf("%s \n", *i.StackId)
			input.Accounts = append(input.Accounts, *i.Account)
			input.Regions = append(input.Regions, *i.Region)
		}
	}

	res, err := getClient().DeleteStackInstances(context.Background(), input)
	if err != nil {
		fmt.Print("error occurred while tried to delete instances")
		return err
	}

	if wait {
		for {
			operation, err := getClient().DescribeStackSetOperation(context.Background(), &cloudformation.DescribeStackSetOperationInput{
				OperationId:  res.OperationId,
				StackSetName: &stackSetName,
			})
			if err != nil || operation == nil || operation.StackSetOperation.Status != types.StackSetOperationStatusRunning {
				break
			}
			time.Sleep(time.Second * 2)
		}
	}
	return err
}

// SetTerminationProtection enables or disables termination protection for a stack
func SetTerminationProtection(stackName string, protectionEnabled bool) error {
	// Set termination protection
	_, err := getClient().UpdateTerminationProtection(context.Background(), &cloudformation.UpdateTerminationProtectionInput{
		StackName:                   &stackName,
		EnableTerminationProtection: ptr.Bool(protectionEnabled),
	})

	return err
}

// GetStack returns a cloudformation.Stack representing the named stack
func GetStack(stackName string) (types.Stack, error) {
	// Get the stack properties
	res, err := getClient().DescribeStacks(context.Background(), &cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})
	if err != nil {
		return types.Stack{}, err
	}

	return res.Stacks[0], nil
}

// GetStackSet returns a cloudformation.StackSet
func GetStackSet(stackSetName string) (types.StackSet, error) {
	// Get the stack properties
	res, err := getClient().DescribeStackSet(context.Background(), &cloudformation.DescribeStackSetInput{
		StackSetName: &stackSetName,
	})
	if err != nil {
		return types.StackSet{}, err
	}

	return *res.StackSet, nil
}

// GetStackResources returns a list of the resources in the named stack
func GetStackResources(stackName string) ([]types.StackResource, error) {
	// Get the stack resources
	res, err := getClient().DescribeStackResources(context.Background(), &cloudformation.DescribeStackResourcesInput{
		StackName: &stackName,
	})
	if err != nil {
		return nil, err
	}

	return res.StackResources, nil
}

// GetStackEvents returns all events associated with the named stack
func GetStackEvents(stackName string) ([]types.StackEvent, error) {
	events := make([]types.StackEvent, 0)

	var token *string

	for {
		res, err := getClient().DescribeStackEvents(context.Background(), &cloudformation.DescribeStackEventsInput{
			NextToken: token,
			StackName: &stackName,
		})

		if err != nil {
			return events, err
		}

		events = append(events, res.StackEvents...)

		if res.NextToken == nil {
			break
		}

		token = res.NextToken
	}

	return events, nil
}

// CreateChangeSet creates a changeset
func CreateChangeSet(template cft.Template, params []types.Parameter, tags map[string]string, stackName string, roleArn string) (string, error) {
	templateBody, err := checkTemplate(template)
	if err != nil {
		return "", err
	}

	changeSetType := "CREATE"

	exists, err := StackExists(stackName)
	if err != nil {
		return "", err
	}

	if exists {
		changeSetType = "UPDATE"
	}

	changeSetName := stackName + "-" + fmt.Sprint(time.Now().Unix())

	input := &cloudformation.CreateChangeSetInput{
		ChangeSetType:       types.ChangeSetType(changeSetType),
		ChangeSetName:       ptr.String(changeSetName),
		StackName:           ptr.String(stackName),
		Tags:                MakeTags(tags),
		IncludeNestedStacks: ptr.Bool(true),
		Parameters:          params,
		Capabilities: []types.Capability{
			"CAPABILITY_NAMED_IAM",
			"CAPABILITY_AUTO_EXPAND",
		},
	}

	if roleArn != "" {
		input.RoleARN = ptr.String(roleArn)
	}

	if strings.HasPrefix(templateBody, "http://") {
		input.TemplateURL = ptr.String(templateBody)
	} else {
		input.TemplateBody = ptr.String(templateBody)
	}

	_, err = getClient().CreateChangeSet(context.Background(), input)
	if err != nil {
		return changeSetName, err
	}

	for {
		res, err := getClient().DescribeChangeSet(context.Background(), &cloudformation.DescribeChangeSetInput{
			ChangeSetName: &changeSetName,
			StackName:     &stackName,
		})
		if err != nil {
			return changeSetName, err
		}

		status := string(res.Status)
		config.Debugf("ChangeSet status: %s", status)

		if status == "FAILED" {
			return changeSetName, errors.New(ptr.ToString(res.StatusReason))
		}

		if strings.HasSuffix(status, "_COMPLETE") {
			break
		}

		time.Sleep(time.Second * 2)
	}

	return changeSetName, nil
}

// GetChangeSet returns the named changeset
func GetChangeSet(stackName, changeSetName string) (*cloudformation.DescribeChangeSetOutput, error) {
	input := &cloudformation.DescribeChangeSetInput{
		ChangeSetName: ptr.String(changeSetName),
	}

	// Stack name is optional
	if stackName != "" {
		input.StackName = ptr.String(stackName)
	}

	return getClient().DescribeChangeSet(context.Background(), input)
}

// CreateStackSet creates stack set
func CreateStackSet(c StackSetConfig) error {

	templateBody, err := checkTemplate(c.Template)
	if err != nil {
		return errors.New("error occured while extracting template body")
	}

	_, err = GetStackSet(*c.StackSetName)
	if err == nil {
		return errors.New("can't create stack set. It already exists")
	}

	input := &cloudformation.CreateStackSetInput{
		StackSetName:          c.StackSetName,
		Parameters:            c.Parameters,
		Tags:                  c.Tags,
		Capabilities:          c.Capabilities,
		Description:           c.Description,
		AdministrationRoleARN: c.AdministrationRoleARN,
		AutoDeployment:        c.AutoDeployment,
		CallAs:                c.CallAs,
		ExecutionRoleName:     c.ExecutionRoleName,
		ManagedExecution:      c.ManagedExecution,
		PermissionModel:       c.PermissionModel,
	}

	if strings.HasPrefix(templateBody, "http://") {
		input.TemplateURL = ptr.String(templateBody)
	} else {
		input.TemplateBody = ptr.String(templateBody)
	}

	res, err := getClient().CreateStackSet(context.Background(), input)

	config.Debugf("Stack set create operation result: \n%s\n", PrettyPrint(res))
	return err
}

func PrettyPrint(i interface{}) string { //TODO
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func CreateStackSetInstances(stackSetInstancesConfig StackSetInstancesConfig) error {

	// input := &cloudformation.CreateStackInstancesInput{ //TODO
	// 	StackSetName: &StackSetName,
	// 	Regions:      []string{"us-east-1"},
	// 	Accounts:     []string{"xxx"},
	// }

	// res, err := getClient().CreateStackInstances(context.Background(), input)

	// config.Debugf("%+v", res)
	// return err
	return nil
}

// ExecuteChangeSet executes the named changeset
func ExecuteChangeSet(stackName, changeSetName string, disableRollback bool) error {
	_, err := getClient().ExecuteChangeSet(context.Background(), &cloudformation.ExecuteChangeSetInput{
		ChangeSetName:   &changeSetName,
		StackName:       &stackName,
		DisableRollback: &disableRollback,
	})

	return err
}

// DeleteChangeSet deletes the named changeset
func DeleteChangeSet(stackName, changeSetName string) error {
	_, err := getClient().DeleteChangeSet(context.Background(), &cloudformation.DeleteChangeSetInput{
		ChangeSetName: &changeSetName,
		StackName:     &stackName,
	})

	return err
}

// WaitUntilStackExists pauses execution until the named stack exists
func WaitUntilStackExists(stackName string) error {
	for {
		_, err := getClient().DescribeStacks(context.Background(), &cloudformation.DescribeStacksInput{
			StackName: ptr.String(stackName),
		})

		if err == nil {
			break
		}

		var apiErr = &smithy.GenericAPIError{}
		if !errors.As(err, &apiErr) {
			return err
		}

		time.Sleep(time.Second * 2)
	}

	return nil
}

// WaitUntilStackCreateComplete pauses execution until the stack is completed (or fails)
func WaitUntilStackCreateComplete(stackName string) error {
	for {
		res, err := getClient().DescribeStacks(context.Background(), &cloudformation.DescribeStacksInput{
			StackName: ptr.String(stackName),
		})

		if err != nil {
			return err
		}

		if len(res.Stacks) != 1 {
			return errors.New("stack not found")
		}

		stack := res.Stacks[0]

		status := string(stack.StackStatus)
		if strings.HasSuffix(status, "_COMPLETE") || strings.HasSuffix(status, "_FAILED") {
			break
		}

		time.Sleep(time.Second * 2)
	}

	return nil
}
