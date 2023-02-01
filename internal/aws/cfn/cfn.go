//go:build !func_test

package cfn

import (
	"context"
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
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
)

var liveStatuses = []types.StackStatus{
	"CREATE_COMPLETE",
	"CREATE_IN_PROGRESS",
	"CREATE_FAILED",
	"DELETE_FAILED",
	"DELETE_IN_PROGRESS",
	"REVIEW_IN_PROGRESS",
	"ROLLBACK_COMPLETE",
	"ROLLBACK_FAILED",
	"ROLLBACK_IN_PROGRESS",
	"UPDATE_COMPLETE",
	"UPDATE_COMPLETE_CLEANUP_IN_PROGRESS",
	"UPDATE_FAILED",
	"UPDATE_IN_PROGRESS",
	"UPDATE_ROLLBACK_COMPLETE",
	"UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS",
	"UPDATE_ROLLBACK_FAILED",
	"UPDATE_ROLLBACK_IN_PROGRESS",
	"IMPORT_IN_PROGRESS",
	"IMPORT_COMPLETE",
	"IMPORT_ROLLBACK_IN_PROGRESS",
	"IMPORT_ROLLBACK_FAILED",
	"IMPORT_ROLLBACK_COMPLETE",
}

const WAIT_PERIOD_IN_SECONDS = 2

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
func DeleteStack(stackName string, roleArn string) error {
	input := &cloudformation.DeleteStackInput{
		StackName: &stackName,
	}

	// roleArn is optional
	if roleArn != "" {
		input.RoleARN = ptr.String(roleArn)
	}

	_, err := getClient().DeleteStack(context.Background(), input)

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
func DeleteAllStackSetInstances(stackSetName string, wait bool, retainStacks bool) error {
	instances, err := ListStackSetInstances(stackSetName)
	if err != nil {
		fmt.Printf("Could not fetch instances for stack set '%s'", stackSetName)
		return err
	}
	accounts := []string{}
	regions := []string{}
	for _, i := range instances {
		if i.StackInstanceStatus.DetailedStatus != types.StackInstanceDetailedStatusRunning { //TODO: do we need to skipp RUNNING only?
			accounts = append(accounts, *i.Account)
			regions = append(regions, *i.Region)
		}
	}
	return DeleteStackSetInstances(stackSetName, accounts, regions, wait, retainStacks)
}

// DeleteStackSetInstances deletes instances for a given stack set in specified accounts and regions
func DeleteStackSetInstances(stackSetName string, accounts []string, regions []string, wait bool, retainStacks bool) error {
	_, err := GetStackSet(stackSetName)
	if err != nil {
		fmt.Printf("Could not find stack set '%s'", stackSetName)
		return err
	}

	var input = &cloudformation.DeleteStackInstancesInput{
		Accounts:     UniqueStrings(accounts),
		Regions:      UniqueStrings(regions),
		RetainStacks: retainStacks,
		StackSetName: &stackSetName,
	}

	res, err := getClient().DeleteStackInstances(context.Background(), input)
	spinner.Pause()
	if err != nil {
		fmt.Print("error occurred while tried to delete instances")
		return err
	}
	fmt.Printf("Submitted DELETE instances operation with ID: %s\n", *res.OperationId)
	spinner.Resume()
	if wait {
		WaitUntilStackSetOperationCompleted(*res.OperationId, stackSetName)
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
func GetStackSet(stackSetName string) (*types.StackSet, error) {
	// Get the stack properties
	res, err := getClient().DescribeStackSet(context.Background(), &cloudformation.DescribeStackSetInput{
		StackSetName: &stackSetName,
	})
	if err != nil {
		return nil, err
	}

	return res.StackSet, nil
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

		time.Sleep(time.Second * WAIT_PERIOD_IN_SECONDS)
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
func CreateStackSet(conf StackSetConfig) (*string, error) {

	templateBody, err := checkTemplate(conf.Template)
	if err != nil {
		return nil, errors.New("error occured while extracting template body")
	}

	_, err = GetStackSet(conf.StackSetName)
	if err == nil {
		return nil, errors.New("can't create stack set. It already exists")
	}

	input := &cloudformation.CreateStackSetInput{
		StackSetName:          &conf.StackSetName,
		Parameters:            conf.Parameters,
		Tags:                  conf.Tags,
		Capabilities:          conf.Capabilities,
		Description:           conf.Description,
		AdministrationRoleARN: conf.AdministrationRoleARN,
		AutoDeployment:        conf.AutoDeployment,
		CallAs:                conf.CallAs,
		ExecutionRoleName:     conf.ExecutionRoleName,
		ManagedExecution:      conf.ManagedExecution,
		PermissionModel:       conf.PermissionModel,
	}

	if strings.HasPrefix(templateBody, "http://") {
		input.TemplateURL = ptr.String(templateBody)
	} else {
		input.TemplateBody = ptr.String(templateBody)
	}

	res, err := getClient().CreateStackSet(context.Background(), input)

	return res.StackSetId, err
}

// UpdateStackSet updates stack set and its instances
func UpdateStackSet(conf StackSetConfig, instanceConf StackSetInstancesConfig, wait bool) error {

	templateBody, err := checkTemplate(conf.Template)
	if err != nil {
		return errors.New("error occured while extracting template body")
	}

	_, err = GetStackSet(conf.StackSetName)
	if err != nil {
		return errors.New("can't update stack set. It does not exists or it is in a wrong state")
	}

	input := &cloudformation.UpdateStackSetInput{
		StackSetName:          &conf.StackSetName,
		Parameters:            conf.Parameters,
		Tags:                  conf.Tags,
		Capabilities:          conf.Capabilities,
		Description:           conf.Description,
		AdministrationRoleARN: conf.AdministrationRoleARN,
		AutoDeployment:        conf.AutoDeployment,
		CallAs:                conf.CallAs,
		ExecutionRoleName:     conf.ExecutionRoleName,
		ManagedExecution:      conf.ManagedExecution,
		PermissionModel:       conf.PermissionModel,
		// instance configuration
		Accounts:             instanceConf.Accounts,
		Regions:              instanceConf.Regions,
		DeploymentTargets:    instanceConf.DeploymentTargets,
		OperationPreferences: instanceConf.OperationPreferences,
	}

	if strings.HasPrefix(templateBody, "http://") {
		input.TemplateURL = ptr.String(templateBody)
	} else {
		input.TemplateBody = ptr.String(templateBody)
	}

	spinner.Pause()
	if len(input.Accounts) == 0 {
		fmt.Println("Updating stack set instances in all previously deployed accounts and regions")
	} else {
		fmt.Printf("Updating stack set instances in...\naccounts: %+v\nregions: %+v\n", input.Accounts, input.Regions)
	}
	spinner.Resume()

	res, err := getClient().UpdateStackSet(context.Background(), input)

	config.Debugf("Update stack instances API result:\n%s", format.PrettyPrint(res))
	if err != nil {
		return err
	}

	spinner.Pause()
	fmt.Printf("Submitted UPDATE stack set operation with ID: %s\n", *res.OperationId)
	spinner.Resume()

	if err != nil {
		return err
	}

	if wait {
		err = WaitUntilStackSetOperationCompleted(*res.OperationId, conf.StackSetName)
	}
	return err
}

// AddStackSetInstances adds instances to a stack set
func AddStackSetInstances(conf StackSetConfig, instanceConf StackSetInstancesConfig, wait bool) error {

	_, err := GetStackSet(conf.StackSetName)
	if err != nil {
		return errors.New("can't update stack set. It does not exists or it is in a wrong state")
	}

	spinner.Pause()
	if len(instanceConf.Accounts) == 0 || len(instanceConf.Regions) == 0 {
		return errors.New("can't update stack set. Account(s) and region(s) must be provided")
	} else {
		fmt.Printf("Adding stack set instances in...\naccounts: %+v\nregions: %+v\n", instanceConf.Accounts, instanceConf.Regions)
	}
	spinner.Resume()

	input := &cloudformation.CreateStackInstancesInput{
		StackSetName:         &conf.StackSetName,
		Accounts:             instanceConf.Accounts,
		Regions:              instanceConf.Regions,
		DeploymentTargets:    instanceConf.DeploymentTargets,
		OperationPreferences: instanceConf.OperationPreferences,
		CallAs:               conf.CallAs,
	}

	res, err := getClient().CreateStackInstances(context.Background(), input)

	config.Debugf("CreateStackInstances API result:\n%s", format.PrettyPrint(res))
	if err != nil {
		return errors.New("error occurred durin stack set update")
	}

	spinner.Pause()
	fmt.Printf("Submitted CREATE stack set instance(s) operation with ID: %s\n", *res.OperationId)
	spinner.Resume()

	if err != nil {
		return err
	}

	if wait {
		err = WaitUntilStackSetOperationCompleted(*res.OperationId, conf.StackSetName)
	}
	return err
}

func CreateStackSetInstances(conf StackSetInstancesConfig, wait bool) error {

	input := &cloudformation.CreateStackInstancesInput{
		StackSetName:         &conf.StackSetName,
		Regions:              conf.Regions,
		Accounts:             conf.Accounts,
		DeploymentTargets:    conf.DeploymentTargets,
		CallAs:               conf.CallAs,
		OperationPreferences: conf.OperationPreferences,
	}

	res, err := getClient().CreateStackInstances(context.Background(), input)
	config.Debugf("Create stack instances API result:\n%s", format.PrettyPrint(res))
	if err != nil {
		fmt.Println("error occurred durin stack set instance(s) deployment ")
		return err
	}

	spinner.Pause()
	fmt.Printf("Submitted CREATE instances operation with ID: %s\n", *res.OperationId)
	spinner.Resume()

	if wait {
		WaitUntilStackSetOperationCompleted(*res.OperationId, conf.StackSetName)
	}

	return err
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

		time.Sleep(time.Second * WAIT_PERIOD_IN_SECONDS)
	}

	return nil
}

func WaitUntilStackSetOperationCompleted(operationId string, stacksetName string) error {
	var operation *cloudformation.DescribeStackSetOperationOutput
	var err error
	for {
		operation, err = getClient().DescribeStackSetOperation(context.Background(), &cloudformation.DescribeStackSetOperationInput{
			OperationId:  &operationId,
			StackSetName: &stacksetName,
		})
		if err != nil || operation == nil ||
			operation.StackSetOperation.Status == types.StackSetOperationStatusStopped ||
			operation.StackSetOperation.Status == types.StackSetOperationStatusSucceeded ||
			operation.StackSetOperation.Status == types.StackSetOperationStatusFailed {
			break
		}

		time.Sleep(time.Second * WAIT_PERIOD_IN_SECONDS)
	}
	if err == nil && operation != nil {
		spinner.Pause()
		fmt.Printf("Stack set operation resulted with state: %s\n", operation.StackSetOperation.Status)
		spinner.Resume()
	}
	return err
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

		time.Sleep(time.Second * WAIT_PERIOD_IN_SECONDS)
	}

	return nil
}
