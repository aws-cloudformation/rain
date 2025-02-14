package cfn

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/smithy-go"
	"gopkg.in/yaml.v3"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/ccapi"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/dc"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/aws-cloudformation/rain/plugins/deployconfig"
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

const WaitPeriodInSeconds = 2

var Schemas map[string]string

//go:embed all-types.txt
var AllTypes string

//go:embed schemas
var schemaFiles embed.FS

func checkTemplate(template cft.Template) (string, error) {
	templateBody := format.String(template, format.Options{})

	// Max template size is 1MB
	if len(templateBody) > (1024 * 1024) {
		return "", fmt.Errorf("template is too large to deploy")
	}

	if len(templateBody) > 51200 {
		config.Debugf("Template is too large to deploy directly; uploading to S3.")

		bucket := s3.RainBucket(false)

		key, err := s3.Upload(bucket, []byte(templateBody), "")
		region := aws.Config().Region
		if strings.HasPrefix(region, "cn-") {
			return fmt.Sprintf("https://%s.s3.%s.amazonaws.com.cn/%s", bucket, region, key), err
		} else {
			return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, key), err
		}
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

// ListChangeSets lists the active change sets associated with a stack
func ListChangeSets(stackName string) ([]types.ChangeSetSummary, error) {
	var token *string
	retval := make([]types.ChangeSetSummary, 0)
	for {
		res, err := getClient().ListChangeSets(context.Background(), &cloudformation.ListChangeSetsInput{
			StackName: &stackName,
			NextToken: token,
		})

		if err != nil {
			return retval, nil
		}

		retval = append(retval, res.Summaries...)

		if res.NextToken == nil {
			break
		}

		token = res.NextToken
	}

	return retval, nil

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

// GetStackOutputs returns an array of Output values from a single deployed stack
func GetStackOutputs(stackName string) ([]types.Output, error) {
	stack, err := GetStack(stackName)
	if err != nil {
		return nil, err
	}
	return stack.Outputs, nil
}

// GetStackResource gets a single deployed stack resource
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

type ChangeSetContext struct {
	Template  cft.Template
	Params    []types.Parameter
	Tags      map[string]string
	StackName string

	// ChangeSetName is optional, if "" is set, the name will be the stack name plus a timestamp
	ChangeSetName string
	RoleArn       string

	// Whether or not to include nested stacks in the change set
	IncludeNested bool
}

// CreateChangeSet creates a changeset
func CreateChangeSet(ctx *ChangeSetContext) (string, error) {

	template := ctx.Template
	params := ctx.Params
	tags := ctx.Tags
	stackName := ctx.StackName
	changeSetName := ctx.ChangeSetName
	roleArn := ctx.RoleArn

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

	if changeSetName == "" {
		changeSetName = stackName + "-" + fmt.Sprint(time.Now().Unix())
	}

	input := &cloudformation.CreateChangeSetInput{
		ChangeSetType:       types.ChangeSetType(changeSetType),
		ChangeSetName:       ptr.String(changeSetName),
		StackName:           ptr.String(stackName),
		Tags:                dc.MakeTags(tags),
		IncludeNestedStacks: ptr.Bool(ctx.IncludeNested),
		Parameters:          params,
		Capabilities: []types.Capability{
			"CAPABILITY_NAMED_IAM",
			"CAPABILITY_AUTO_EXPAND",
		},
	}

	if roleArn != "" {
		input.RoleARN = ptr.String(roleArn)
	}

	if strings.HasPrefix(templateBody, "http") {
		input.TemplateURL = ptr.String(templateBody)
	} else {
		input.TemplateBody = ptr.String(templateBody)
		config.Debugf("About to create changeset with body:\n%s", templateBody)
		for _, param := range params {
			var k, v string
			if param.ParameterKey != nil {
				k = *param.ParameterKey
			}
			if param.ParameterValue != nil {
				v = *param.ParameterValue
			}
			config.Debugf("Parameter Key: %s, Value: %s", k, v)
		}
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

		time.Sleep(time.Second * WaitPeriodInSeconds)
	}

	return changeSetName, nil
}

// GetChangeSet returns the named changeset
func GetChangeSet(stackName, changeSetName string) (*cloudformation.DescribeChangeSetOutput, error) {
	input := &cloudformation.DescribeChangeSetInput{
		ChangeSetName:         ptr.String(changeSetName),
		IncludePropertyValues: ptr.Bool(true),
	}

	// Stack name is optional
	if stackName != "" {
		input.StackName = ptr.String(stackName)
	}

	return getClient().DescribeChangeSet(context.Background(), input)
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

		time.Sleep(time.Second * WaitPeriodInSeconds)
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

		time.Sleep(time.Second * WaitPeriodInSeconds)
	}

	return nil
}

// GetTypeSchema gets the schema for a CloudFormation resource type
func GetTypeSchema(name string, cacheUsage ResourceCacheUsage) (string, error) {

	// Check for a schema in memory
	schema, exists := Schemas[name]
	if exists && cacheUsage != DoNotUseCache {
		return schema, nil
	}

	// Look in the embedded file system next
	if cacheUsage != DoNotUseCache {
		path := strings.Replace(name, "::", "/", -1)
		path = strings.ToLower(path)
		path = "schemas/" + path + ".json"
		b, err := schemaFiles.ReadFile(path)
		if err == nil {
			s := string(b)
			Schemas[name] = s
			return s, nil
		} else {
			config.Debugf("unable to read schema from path %s: %v", path, err)
		}
	}

	if cacheUsage == OnlyUseCache {
		return "", errors.New("cacheUsage is OnlyUseCache")
	}

	// Go ahead and download the schema from the registry
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

// IsCCAPI returns true if the type is fully supported by CCAPI
func IsCCAPI(name string) (bool, error) {
	res, err := getClient().DescribeType(context.Background(), &cloudformation.DescribeTypeInput{
		Type: "RESOURCE", TypeName: &name,
	})
	if err != nil {
		config.Debugf("SDK error: %v", err)
		return false, err
	}
	// Check 3rd party types to see if they have been activated in this region
	if res.IsActivated != nil && !*res.IsActivated {
		return false, nil
	}

	// Make sure it's fully mutable
	if res.ProvisioningType != types.ProvisioningTypeFullyMutable {
		return false, nil
	}

	return true, nil
}

// GetTypePermissions gets the list of actions required to invoke a CloudFormation handler
func GetTypePermissions(name string, handlerVerb string) ([]string, error) {

	// Get the schema, checking to see if we cached it
	schema, err := GetTypeSchema(name, UseCacheNormally)
	if err != nil {
		return nil, err
	}

	// Parse the schema and return the array of actions
	var result map[string]any
	err = json.Unmarshal([]byte(schema), &result)
	if err != nil {
		return nil, err
	}
	/* "handlers": {
	   "create": {
	       "permissions": [
	           "s3:CreateBucket",
	           "s3:PutBucketTagging",

	*/

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
	handlers := handlerMap.(map[string]any)
	handlerVerbMap, exists := handlers[handlerVerb]
	if !exists {
		// Some resources can't be updated, for example
		return retval, nil
	}
	handler := handlerVerbMap.(map[string]any)
	permissions := handler["permissions"].([]interface{})
	for _, p := range permissions {
		if p == "iam:PassRole" {
			// This will fail even for admin roles, and is not actually necessary
			// to create resources like buckets, despite being in the schema
			continue
		}
		retval = append(retval, fmt.Sprintf("%v", p))
	}
	return retval, nil
}

// GetTypeIdentifier gets the primaryIdentifier of a resource type from the schema
func GetTypeIdentifier(name string) ([]string, error) {
	schema, err := GetTypeSchema(name, UseCacheNormally)
	if err != nil {
		return nil, err
	}
	if schema == "" {
		return nil, errors.New("schema is empty")
	}

	var result map[string]any
	err = json.Unmarshal([]byte(schema), &result)
	if err != nil {
		return nil, err
	}

	piNode, exists := result["primaryIdentifier"]
	if !exists {
		// The schema does not have a primary identifier.
		// TODO
		return nil, errors.New("no primary identifier")
	} else {
		pi := piNode.([]interface{})
		retval := make([]string, 0)
		for _, pid := range pi {
			retval = append(retval, strings.Replace(fmt.Sprintf("%v", pid), "/properties/", "", 1))
		}
		return retval, nil
	}
}

// GetPrimaryIdentifierValues gets the values specified for primary identifiers in the template.
// The return value will only have values if they are set.
// TODO: Use ccapi to look at the deployed resource model for updates
func GetPrimaryIdentifierValues(
	primaryIdentifier []string,
	resource *yaml.Node,
	template *yaml.Node,
	dc *deployconfig.DeployConfig) []string {

	piValues := make([]string, 0)

	_, props, _ := s11n.GetMapValue(resource, "Properties")
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
					piValues = append(piValues, val)
				} else {
					// Likely a !Ref or !Sub
					if content.Kind == yaml.MappingNode {
						if content.Content[0].Value == "Ref" && content.Content[1].Kind == yaml.ScalarNode {
							val, err := resolveRef(content.Content[1].Value, template, dc)
							if err == nil {
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
// TODO: ccdeploy.resolve does this better
// TODO: Is this dead code now? We resolve refs early in forecast.
//
//	What else uses this?
func resolveRef(name string, template *yaml.Node, dc *deployconfig.DeployConfig) (string, error) {
	_, params, _ := s11n.GetMapValue(template.Content[0], "Parameters")
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
	dc *deployconfig.DeployConfig) bool {

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

type ResourceCacheUsage int

const (
	OnlyUseCache     ResourceCacheUsage = 1
	DoNotUseCache    ResourceCacheUsage = 2
	UseCacheNormally ResourceCacheUsage = 3
)

// ListResourceTypes lists all live registry resource types
func ListResourceTypes(cacheUsage ResourceCacheUsage) ([]string, error) {

	if cacheUsage != DoNotUseCache {
		return strings.Split(AllTypes, "\n"), nil
	}

	input := &cloudformation.ListTypesInput{
		DeprecatedStatus: types.DeprecatedStatusLive,
		Type:             types.RegistryTypeResource,
	}

	retval := make([]string, 0)
	vis := []types.Visibility{types.VisibilityPublic, types.VisibilityPrivate}

	for _, v := range vis {
		hasMore := true
		for hasMore {
			input.Visibility = v
			res, err := getClient().ListTypes(context.Background(), input)
			if err != nil {
				return retval, err
			}

			for _, s := range res.TypeSummaries {
				retval = append(retval, *s.TypeName)
			}

			if res.NextToken != nil {
				hasMore = true
				input.NextToken = res.NextToken
			} else {
				hasMore = false
				input.NextToken = nil
			}
		}
	}

	return retval, nil

}

func init() {
	Schemas = make(map[string]string)
}
