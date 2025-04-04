package cfn

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
)

// ListStackSets returns a list of all existing stack sets
func ListStackSets(delegateAdmin bool) ([]types.StackSetSummary, error) {
	stackSets := make([]types.StackSetSummary, 0)

	var token *string

	callas := types.CallAsSelf
	if delegateAdmin {
		callas = types.CallAsDelegatedAdmin
	}

	for {
		res, err := getClient().ListStackSets(context.Background(), &cloudformation.ListStackSetsInput{
			NextToken: token,
			CallAs:    callas,
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
func ListStackSetInstances(stackSetName string, delegatedAdmin bool) ([]types.StackInstanceSummary, error) {
	instances := make([]types.StackInstanceSummary, 0)
	var token *string

	callas := types.CallAsSelf
	if delegatedAdmin {
		callas = types.CallAsDelegatedAdmin
	}

	for {
		res, err := getClient().ListStackInstances(context.Background(), &cloudformation.ListStackInstancesInput{
			NextToken:    token,
			StackSetName: &stackSetName,
			CallAs:       callas,
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
func ListLast10StackSetOperations(stackSetName string, delegatedAdmin bool) ([]types.StackSetOperationSummary, error) {
	stackOps := make([]types.StackSetOperationSummary, 0)

	callas := types.CallAsSelf
	if delegatedAdmin {
		callas = types.CallAsDelegatedAdmin
	}
	res, err := getClient().ListStackSetOperations(context.Background(), &cloudformation.ListStackSetOperationsInput{
		MaxResults:   ptr.Int32(10),
		StackSetName: &stackSetName,
		CallAs:       callas,
	})

	if err != nil {
		return stackOps, err
	}

	stackOps = append(stackOps, res.Summaries...)
	return stackOps, nil
}

// GetStackSetOperationsResult  returns an operation result for a given stack sets operation id
func GetStackSetOperationsResult(stackSetName *string, operationId *string, delegatedAdmin bool) (*types.StackSetOperationResultSummary, error) {
	callas := types.CallAsSelf
	if delegatedAdmin {
		callas = types.CallAsDelegatedAdmin
	}
	res, err := getClient().ListStackSetOperationResults(context.Background(), &cloudformation.ListStackSetOperationResultsInput{
		MaxResults:   ptr.Int32(1),
		OperationId:  operationId,
		StackSetName: stackSetName,
		CallAs:       callas,
	})

	if err == nil && res != nil && len(res.Summaries) == 1 {
		return &res.Summaries[0], err
	}
	return nil, nil
}

// DeleteStackSet deletes a stack set
func DeleteStackSet(stackSetName string, delegatedAdmin bool) error {
	callas := types.CallAsSelf
	if delegatedAdmin {
		callas = types.CallAsDelegatedAdmin
	}
	_, err := getClient().DeleteStackSet(context.Background(), &cloudformation.DeleteStackSetInput{
		StackSetName: &stackSetName,
		CallAs:       callas,
	})

	return err
}

// DeleteAllStackSetInstances deletes all instances for a given stack set
func DeleteAllStackSetInstances(stackSetName string, wait bool, retainStacks bool, delegatedAdmin bool) error {
	instances, err := ListStackSetInstances(stackSetName, delegatedAdmin)
	if err != nil {
		fmt.Printf("Could not fetch instances for stack set '%s'", stackSetName)
		return err
	}
	accounts := make([]string, 0)
	regions := make([]string, 0)
	for _, i := range instances {
		if i.StackInstanceStatus.DetailedStatus != types.StackInstanceDetailedStatusRunning { //TODO: do we need to skipp RUNNING only?
			accounts = append(accounts, *i.Account)
			regions = append(regions, *i.Region)
		}
	}
	return DeleteStackSetInstances(stackSetName, accounts, regions, wait, retainStacks, delegatedAdmin)
}

// DeleteStackSetInstances deletes instances for a given stack set in specified accounts and regions
func DeleteStackSetInstances(stackSetName string, accounts []string, regions []string, wait bool, retainStacks bool, delegatedAdmin bool) error {
	_, err := GetStackSet(stackSetName, delegatedAdmin)
	if err != nil {
		fmt.Printf("Could not find stack set '%s'", stackSetName)
		return err
	}

	callas := types.CallAsSelf
	if delegatedAdmin {
		callas = types.CallAsDelegatedAdmin
	}
	var input = &cloudformation.DeleteStackInstancesInput{
		Accounts:     UniqueStrings(accounts),
		Regions:      UniqueStrings(regions),
		RetainStacks: &retainStacks,
		StackSetName: &stackSetName,
		CallAs:       callas,
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
		err := WaitUntilStackSetOperationCompleted(*res.OperationId, stackSetName)
		if err != nil {
			return err
		}
	}
	return err
}

// GetStackSet returns a cloudformation.StackSet
func GetStackSet(stackSetName string, delegatedAdmin bool) (*types.StackSet, error) {
	// Get the stack properties

	callas := types.CallAsSelf
	if delegatedAdmin {
		callas = types.CallAsDelegatedAdmin
	}

	res, err := getClient().DescribeStackSet(context.Background(), &cloudformation.DescribeStackSetInput{
		StackSetName: &stackSetName,
		CallAs:       callas,
	})
	if err != nil {
		return nil, err
	}

	return res.StackSet, nil
}

// CreateStackSet creates stack set
func CreateStackSet(conf StackSetConfig) (*string, error) {

	templateBody, err := checkTemplate(conf.Template)
	if err != nil {
		return nil, errors.New("error occurred while extracting template body")
	}

	_, err = GetStackSet(conf.StackSetName, conf.CallAs == types.CallAsDelegatedAdmin)
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

	if strings.HasPrefix(templateBody, "https://") {
		input.TemplateURL = ptr.String(templateBody)
	} else {
		input.TemplateBody = ptr.String(templateBody)
	}

	res, err := getClient().CreateStackSet(context.Background(), input)

	if err != nil {
		return nil, err
	}
	return res.StackSetId, err
}

// UpdateStackSet updates stack set and its instances
func UpdateStackSet(conf StackSetConfig, instanceConf StackSetInstancesConfig, wait bool) error {

	templateBody, err := checkTemplate(conf.Template)
	if err != nil {
		return errors.New("error occurred while extracting template body")
	}

	_, err = GetStackSet(conf.StackSetName, conf.CallAs == types.CallAsDelegatedAdmin)
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

	if strings.HasPrefix(templateBody, "http") {
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

	_, err := GetStackSet(conf.StackSetName, conf.CallAs == types.CallAsDelegatedAdmin)
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
		err := WaitUntilStackSetOperationCompleted(*res.OperationId, conf.StackSetName)
		if err != nil {
			return err
		}
	}

	return err
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

		time.Sleep(time.Second * WaitPeriodInSeconds)
	}
	if err == nil && operation != nil {
		spinner.Pause()
		fmt.Printf("Stack set operation resulted with state: %s\n", operation.StackSetOperation.Status)
		spinner.Resume()
	}
	return err
}
