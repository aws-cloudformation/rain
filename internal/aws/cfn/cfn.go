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
	"gopkg.in/yaml.v3"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/ccapi"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/dc"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
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

var Schemas map[string]string

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
		RetainStacks: &retainStacks,
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

// Get a single deployed stack resource
func GetStackResource(stackName string, logicalId string) (*types.StackResourceDetail, error) {
	res, err := getClient().DescribeStackResource(context.Background(),
		&cloudformation.DescribeStackResourceInput{
			StackName:         &stackName,
			LogicalResourceId: &logicalId,
		})
	if err != nil {
		return nil, err
	}
	return res.StackResourceDetail, nil
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
		Tags:                dc.MakeTags(tags),
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

	if err != nil {
		return nil, err
	}
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

// Get the schema for a CloudFormation resource type
func GetTypeSchema(name string) (string, error) {
	schema, exists := Schemas[name]
	if exists {
		config.Debugf("Already downloaded schema for %v", name)
		return schema, nil
	} else {
		config.Debugf("Downloading schema for %v", name)
		res, err := getClient().DescribeType(context.Background(), &cloudformation.DescribeTypeInput{
			Type: "RESOURCE", TypeName: &name,
		})
		if err != nil {
			config.Debugf("GetTypeSchema SDK error: %v", err)
			return "", err
		}
		Schemas[name] = *res.Schema
		return *res.Schema, nil
	}
}

// Get the list of actions required to invoke a CloudFormation handler
func GetTypePermissions(name string, handlerVerb string) ([]string, error) {

	// Get the schema, checking to see if we cached it
	schema, err := GetTypeSchema(name)
	if err != nil {
		return nil, err
	}

	// Parse the schema and return the array of actions
	var result map[string]any
	json.Unmarshal([]byte(schema), &result)
	/* "handlers": {
	   "create": {
	       "permissions": [
	           "s3:CreateBucket",
	           "s3:PutBucketTagging",

	*/

	config.Debugf("GetTypePermissions result: %v", result)

	retval := make([]string, 0)

	handlerMap, exists := result["handlers"]
	if !exists {
		// Resources that have not been fully migrated to the registry won't have this.
		// This is a best guess.. don't think legacy resource permissions are documented anywhere
		// This will become dead code as soon as the permissions are available from the registry.
		if name == "AWS::EC2::Instance" {
			handlerMap = map[string]any{
				"create": map[string]any{
					"permissions": []any{
						"ec2:AttachVolume",
						"ec2:CreateTags",
						"ec2:RunInstances",
						"ec2:StartInstances",
					},
				},
				"read": map[string]any{
					"permissions": []any{
						"ec2:DescribeInstanceAttribute",
						"ec2:DescribeInstanceStatus",
						"ec2:DescribeInstances",
						"ec2:DescribeTags",
					},
				},
				"update": map[string]any{
					"permissions": []any{
						"ec2:AttachVolume",
						"ec2:CreateTags",
						"ec2:DeleteTags",
						"ec2:DescribeInstanceAttribute",
						"ec2:DescribeInstanceStatus",
						"ec2:DescribeInstances",
						"ec2:DescribeTags",
						"ec2:DetachVolume",
						"ec2:ModifyInstanceAttribute",
						"ec2:StartInstances",
						"ec2:StopInstances",
						"ec2:TerminateInstances",
					},
				},
				"delete": map[string]any{
					"permissions": []any{
						"ec2:DeleteTags",
						"ec2:DescribeInstanceAttribute",
						"ec2:DescribeInstanceStatus",
						"ec2:DescribeInstances",
						"ec2:DescribeTags",
						"ec2:DetachVolume",
						"ec2:StopInstances",
						"ec2:TerminateInstances",
					},
				},
			}
		} else if name == "AWS::Lambda::Alias" {
			handlerMap = map[string]any{
				"create": map[string]any{
					"permissions": []any{
						"lambda:CreateAlias",
						"lambda:GetAlias",
						"lambda:GetFunctionConfiguration",
					},
				},
				"read": map[string]any{
					"permissions": []any{
						"lambda:GetAlias",
						"lambda:GetFunctionConfiguration",
					},
				},
				"update": map[string]any{
					"permissions": []any{
						"lambda:CreateAlias",
						"lambda:DeleteAlias",
						"lambda:GetAlias",
						"lambda:GetFunctionConfiguration",
						"lambda:UpdateAlias",
					},
				},
				"delete": map[string]any{
					"permissions": []any{
						"lambda:DeleteAlias",
						"lambda:GetAlias",
						"lambda:GetFunctionConfiguration",
					},
				},
			}
		} else if name == "AWS::Lambda::Version" {
			handlerMap = map[string]any{
				"create": map[string]any{
					"permissions": []any{
						"lambda:GetFunctionConfiguration",
						"lambda:CreateFunction",
						"lambda:GetFunction",
						"lambda:PutFunctionConcurrency",
						"lambda:GetCodeSigningConfig",
						"lambda:GetFunctionCodeSigningConfig",
						"lambda:GetRuntimeManagementConfig",
						"lambda:PutRuntimeManagementConfig",
					},
				},
				"read": map[string]any{
					"permissions": []any{
						"lambda:GetFunctionConfiguration",
						"lambda:GetFunction",
						"lambda:GetFunctionCodeSigningConfig",
					},
				},
				"update": map[string]any{
					"permissions": []any{
						"lambda:GetFunctionConfiguration",
						"lambda:DeleteFunctionConcurrency",
						"lambda:GetFunction",
						"lambda:PutFunctionConcurrency",
						"lambda:ListTags",
						"lambda:TagResource",
						"lambda:UntagResource",
						"lambda:UpdateFunctionConfiguration",
						"lambda:UpdateFunctionCode",
						"lambda:PutFunctionCodeSigningConfig",
						"lambda:DeleteFunctionCodeSigningConfig",
						"lambda:GetCodeSigningConfig",
						"lambda:GetFunctionCodeSigningConfig",
						"lambda:GetRuntimeManagementConfig",
						"lambda:PutRuntimeManagementConfig",
					},
				},
				"delete": map[string]any{
					"permissions": []any{
						"lambda:GetFunctionConfiguration",
						"lambda:DeleteFunction",
					},
				},
			}
		} else {
			// Return an empty array
			config.Debugf("No data on what permissions are required for %v", name)
			return retval, nil
		}
	}
	config.Debugf("handlerMap: %v", handlerMap)
	handlers := handlerMap.(map[string]any)
	handlerVerbMap, exists := handlers[handlerVerb]
	if !exists {
		config.Debugf("handler verb is missing: %v", handlerVerb)
		// Some resources can't be updated, for example
		return retval, nil
	}
	handler := handlerVerbMap.(map[string]any)
	config.Debugf("handler: %v", handler)
	permissions := handler["permissions"].([]interface{})
	config.Debugf("Got permissions for %v %v: %v", name, handlerVerb, permissions)
	for _, p := range permissions {
		if p == "iam:PassRole" {
			// This will fail even for admin roles, and is not actually necessary
			// to create resources like buckets, despite being in the schema
			continue
		}
		retval = append(retval, fmt.Sprintf("%v", p))
	}
	config.Debugf("retval is %v", retval)
	return retval, nil
}

// Get the primaryIdentifier of a resource type from the schema
func GetTypeIdentifier(name string) ([]string, error) {
	schema, err := GetTypeSchema(name)
	if err != nil {
		return nil, err
	}
	if schema == "" {
		return nil, errors.New("schema is empty")
	}

	var result map[string]any
	json.Unmarshal([]byte(schema), &result)

	config.Debugf("GetTypeIdentifier schema for %s: %v", name, result)

	piNode, exists := result["primaryIdentifier"]
	if !exists {
		// The schema does not have a primary identifier.
		// TODO
		config.Debugf("GetTypeIdentifier %v does not have a primaryIdentifier", name)
		return nil, errors.New("no primary identifier")
	} else {
		pi := piNode.([]interface{})
		retval := make([]string, 0)
		for _, pid := range pi {
			retval = append(retval, strings.Replace(fmt.Sprintf("%v", pid), "/properties/", "", 1))
		}
		config.Debugf("GetTypeIdentifier for %v: %v", name, retval)
		return retval, nil
	}
}

// Get the values specified for primary identifiers in the template.
// The return value will only have values if they are set.
func GetPrimaryIdentifierValues(
	primaryIdentifier []string,
	resource *yaml.Node,
	template *yaml.Node,
	dc *dc.DeployConfig) []string {

	piValues := make([]string, 0)

	_, props := s11n.GetMapValue(resource, "Properties")
	if props == nil {
		return piValues
	}
	for _, pi := range primaryIdentifier {
		for i, prop := range props.Content {
			if i%2 != 0 {
				continue
			}
			propName := prop.Value
			if pi == propName {
				content := props.Content[i+1]
				if content.Kind == yaml.ScalarNode {
					val := content.Value
					config.Debugf("pi %v = %v", pi, val)
					piValues = append(piValues, val)
				} else {
					// Likely a !Ref or !Sub
					config.Debugf("PrimaryIdentifier: %v", node.ToJson(content))
					if content.Kind == yaml.MappingNode {
						if content.Content[0].Value == "Ref" && content.Content[1].Kind == yaml.ScalarNode {
							val, err := resolveRef(content.Content[1].Value, template, dc)
							if err == nil {
								config.Debugf("Resolved Ref %v: %v", content.Content[1].Value, val)
								piValues = append(piValues, val)
							} else {
								config.Debugf("%v", err)
							}
						} else {
							config.Debugf("PrimaryIdentifier, unable to resolve %v", content.Content[0].Value)
						}
					}
				}
			}
		}
	}

	return piValues
}

// resolveRef resolves a scalar reference if we have enough information
// Returns "", error if the Ref can't be resolved (not a panic condition)
func resolveRef(name string, template *yaml.Node, dc *dc.DeployConfig) (string, error) {
	_, params := s11n.GetMapValue(template.Content[0], "Parameters")
	config.Debugf("resolveRef params: %v", node.ToJson(params))
	if params != nil {
		for i, param := range params.Content {
			if i%2 != 0 {
				continue
			}
			if param.Kind == yaml.ScalarNode && param.Value == name {
				// Get the value of the parameter from command line args
				config.Debugf("Checking DeployConfig for %v", name)

				for _, param := range dc.Params {
					if *param.ParameterKey == name {
						return *param.ParameterValue, nil
					}
				}
			}
		}
	}

	return "", errors.New("could not resolve Ref")
}

// ResourceAlreadyExists returns true if the resource has all of its primary
// identifiers hard coded into the template, and this is not a stack update,
// and a resource with those identifiers already exists.
func ResourceAlreadyExists(
	typeName string,
	resource *yaml.Node,
	stackExists bool,
	template *yaml.Node,
	dc *dc.DeployConfig) bool {

	if !stackExists {
		primaryIdentifiers, err := GetTypeIdentifier(typeName)
		if err != nil {
			config.Debugf("Unable to get primary identifier for %v: %v", typeName, err)
			return false
		} else {
			config.Debugf("PrimaryIdentifiers: %v", primaryIdentifiers)

			// See if the primary identifier was user-specified in the template
			piValues := GetPrimaryIdentifierValues(primaryIdentifiers, resource, template, dc)
			config.Debugf("piValues: %v", piValues)

			if len(piValues) == len(primaryIdentifiers) {
				// All primary identifiers were specified in the template
				// Ask CCAPI if the resource already exists

				// TODO - Make sure the type is actually supported by CCAPI
				// Something like this:
				// aws cloudformation list-types --type RESOURCE --visibility PUBLIC --provisioning-type FULLY_MUTABLE --max-results 100

				if ccapi.ResourceExists(typeName, piValues) {
					return true
				}
			}
		}
	} else {
		// TODO - Look at the change set for newly added resources
		config.Debugf("Checking change set for new resources")
	}

	return false
}

func init() {
	Schemas = make(map[string]string)
}
