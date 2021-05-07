//+build func_test

package cfn

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
)

type mockStack struct {
	name      string
	template  cft.Template
	stack     types.Stack
	resources []types.StackResource
}

type mockChangeSet struct {
	template  cft.Template
	params    []types.Parameter
	tags      map[string]string
	stackName string
}

type regionConfig struct {
	stacks     map[string]*mockStack
	changeSets map[string]*mockChangeSet
}

var regions = map[string]regionConfig{
	"mock-region-1": {
		stacks:     make(map[string]*mockStack),
		changeSets: make(map[string]*mockChangeSet),
	},
	"mock-region-2": {
		stacks:     make(map[string]*mockStack),
		changeSets: make(map[string]*mockChangeSet),
	},
	"mock-region-3": {
		stacks:     make(map[string]*mockStack),
		changeSets: make(map[string]*mockChangeSet),
	},
}

func region() regionConfig {
	return regions[aws.Config().Region]
}

var noStackErr = errors.New("No such mock stack")
var noChangeSetErr = errors.New("No such mock change set")
var wrongChangeSetErr = errors.New("Mock change set does not match stack name")

var now = time.Date(2010, time.September, 9, 0, 0, 0, 0, time.UTC)

// GetStackTemplate returns the template used to launch the named stack
func GetStackTemplate(stackName string, processed bool) (string, error) {
	if s, ok := region().stacks[stackName]; ok {
		return format.String(s.template, format.Options{}), nil
	}

	return "", noStackErr
}

// StackExists checks whether the named stack currently exists
func StackExists(stackName string) (bool, error) {
	_, ok := region().stacks[stackName]

	return ok, nil
}

// ListStacks returns a list of all existing stacks
func ListStacks() ([]types.StackSummary, error) {
	out := make([]types.StackSummary, 0)

	for _, s := range region().stacks {
		if s.stack.StackStatus != types.StackStatusCreateFailed && s.stack.StackStatus != types.StackStatusDeleteComplete {
			out = append(out, types.StackSummary{
				CreationTime:        s.stack.CreationTime,
				StackName:           s.stack.StackName,
				StackStatus:         s.stack.StackStatus,
				DeletionTime:        s.stack.DeletionTime,
				LastUpdatedTime:     s.stack.LastUpdatedTime,
				ParentId:            s.stack.ParentId,
				RootId:              s.stack.RootId,
				StackId:             s.stack.StackId,
				StackStatusReason:   s.stack.StackStatusReason,
				TemplateDescription: s.stack.Description,
			})
		}
	}

	return out, nil
}

// DeleteStack deletes a stack
func DeleteStack(stackName string) error {
	if s, ok := region().stacks[stackName]; ok {
		s.stack.StackStatus = types.StackStatusDeleteComplete
		return nil
	}

	return noStackErr
}

// SetTerminationProtection enables or disables termination protection for a stack
func SetTerminationProtection(stackName string, protectionEnabled bool) error {
	if s, ok := region().stacks[stackName]; ok {
		s.stack.EnableTerminationProtection = ptr.Bool(true)
		return nil
	}

	return noStackErr
}

// GetStack returns a cloudformation.Stack representing the named stack
func GetStack(stackName string) (types.Stack, error) {
	if s, ok := region().stacks[stackName]; ok {
		return s.stack, nil
	}

	return types.Stack{}, noStackErr
}

// GetStackResources returns a list of the resources in the named stack
func GetStackResources(stackName string) ([]types.StackResource, error) {
	if s, ok := region().stacks[stackName]; ok {
		return s.resources, nil
	}

	return nil, noStackErr
}

// GetStackEvents returns all events associated with the named stack
func GetStackEvents(stackName string) ([]types.StackEvent, error) {
	return []types.StackEvent{
		{
			EventId:              ptr.String("mock event id"),
			StackId:              ptr.String(stackName),
			StackName:            ptr.String(stackName),
			Timestamp:            &now,
			ClientRequestToken:   ptr.String("mock event token"),
			LogicalResourceId:    ptr.String("MockResourceId"),
			PhysicalResourceId:   ptr.String("MockPhysicalId"),
			ResourceProperties:   ptr.String("mock resource properties"),
			ResourceStatus:       types.ResourceStatusCreateInProgress,
			ResourceStatusReason: ptr.String("mock status reason"),
			ResourceType:         ptr.String("Mock::Resource::Type"),
		},
	}, nil
}

// CreateChangeSet creates a changeset
func CreateChangeSet(template cft.Template, params []types.Parameter, tags map[string]string, stackName string) (string, error) {
	name := uuid.New().String()

	region().changeSets[name] = &mockChangeSet{
		template:  template,
		params:    params,
		tags:      tags,
		stackName: stackName,
	}

	return name, nil
}

// GetChangeSet returns the named changeset
func GetChangeSet(stackName, changeSetName string) (*cloudformation.DescribeChangeSetOutput, error) {
	c, ok := region().changeSets[changeSetName]
	if !ok {
		return nil, noChangeSetErr
	}

	if c.stackName != stackName {
		return nil, fmt.Errorf("Mock change set's stack name is not '%s'", stackName)
	}

	return &cloudformation.DescribeChangeSetOutput{
		Capabilities:  []types.Capability{},
		ChangeSetId:   ptr.String(changeSetName),
		ChangeSetName: ptr.String(changeSetName),
		Changes: []types.Change{
			{
				ResourceChange: &types.ResourceChange{
					Action: types.ChangeActionAdd,
					Details: []types.ResourceChangeDetail{
						{
							CausingEntity: ptr.String("mock entity"),
							ChangeSource:  types.ChangeSourceResourceAttribute,
							Evaluation:    types.EvaluationTypeDynamic,
							Target: &types.ResourceTargetDefinition{
								Attribute:          types.ResourceAttributeProperties,
								Name:               ptr.String("mock attribute"),
								RequiresRecreation: types.RequiresRecreationNever,
							},
						},
					},
					LogicalResourceId:  ptr.String("MockResourceId"),
					PhysicalResourceId: ptr.String("MockPhysicalId"),
					Replacement:        types.ReplacementFalse,
					ResourceType:       ptr.String("Mock::Resource::Type"),
					Scope:              []types.ResourceAttribute{},
				},
				Type: types.ChangeTypeResource,
			},
		},
		CreationTime:          &now,
		Description:           ptr.String("Mock change set"),
		ExecutionStatus:       types.ExecutionStatusAvailable,
		NextToken:             nil,
		NotificationARNs:      []string{},
		Parameters:            c.params,
		RollbackConfiguration: &types.RollbackConfiguration{},
		StackId:               ptr.String(stackName),
		StackName:             ptr.String(stackName),
		Status:                types.ChangeSetStatusCreateComplete,
		StatusReason:          ptr.String("Mock status reason"),
		Tags:                  makeTags(c.tags),
	}, nil
}

// ExecuteChangeSet executes the named changeset
func ExecuteChangeSet(stackName, changeSetName string) error {
	c, ok := region().changeSets[changeSetName]
	if !ok {
		return noChangeSetErr
	}

	if c.stackName != stackName {
		return wrongChangeSetErr
	}

	s, ok := region().stacks[stackName]
	if !ok {
		s = &mockStack{
			name:     stackName,
			template: c.template,
		}

		region().stacks[stackName] = s
	}

	s.stack = types.Stack{
		CreationTime:                &now,
		StackName:                   ptr.String(stackName),
		StackStatus:                 types.StackStatusCreateComplete,
		Capabilities:                []types.Capability{},
		ChangeSetId:                 ptr.String(changeSetName),
		Description:                 ptr.String("Mock stack description"),
		DisableRollback:             ptr.Bool(false),
		EnableTerminationProtection: ptr.Bool(false),
		Outputs: []types.Output{
			{
				Description: ptr.String("Mock output description"),
				ExportName:  ptr.String("MockExport"),
				OutputKey:   ptr.String("MockKey"),
				OutputValue: ptr.String("Mock value"),
			},
		},
		Parameters: []types.Parameter{
			{
				ParameterKey:   ptr.String("MockKey"),
				ParameterValue: ptr.String("Mock value"),
			},
		},
		StackId:           ptr.String(stackName),
		StackStatusReason: ptr.String("Mock status reason"),
		Tags:              makeTags(c.tags),
	}

	s.resources = []types.StackResource{
		{
			LogicalResourceId:    ptr.String("MockResourceId"),
			ResourceStatus:       types.ResourceStatusCreateComplete,
			ResourceType:         ptr.String("Mock::Resource::Type"),
			Timestamp:            &now,
			Description:          ptr.String("Mock resource description"),
			PhysicalResourceId:   ptr.String("MockPhysicalId"),
			ResourceStatusReason: ptr.String("Mock status reason"),
			StackId:              ptr.String(stackName),
			StackName:            ptr.String(stackName),
		},
	}

	return nil
}

// DeleteChangeSet deletes the named changeset
func DeleteChangeSet(stackName, changeSetName string) error {
	c, ok := region().changeSets[changeSetName]
	if !ok {
		return noChangeSetErr
	}

	if c.stackName != stackName {
		return wrongChangeSetErr
	}

	delete(region().changeSets, changeSetName)
	return nil
}

// WaitUntilStackExists pauses execution until the named stack exists
func WaitUntilStackExists(stackName string) error {
	if _, ok := region().stacks[stackName]; !ok {
		return noStackErr
	}

	return nil
}

// WaitUntilStackCreateComplete pauses execution until the stack is completed (or fails)
func WaitUntilStackCreateComplete(stackName string) error {
	if _, ok := region().stacks[stackName]; !ok {
		return noStackErr
	}

	return nil
}
