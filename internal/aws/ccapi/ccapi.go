package ccapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/appscode/jsonpatch"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/aws/aws-sdk-go-v2/service/cloudcontrol"
	"github.com/aws/aws-sdk-go-v2/service/cloudcontrol/types"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

func getClient() *cloudcontrol.Client {
	return cloudcontrol.NewFromConfig(aws.Config())
}

// Returns true if the resource already exists
func ResourceExists(typeName string, identifier []string) bool {

	id := ""

	// CCAPI expects the identifier to match the order that the
	// primaryIdentifier is documented in the schema
	for i, idValue := range identifier {
		id += idValue
		if i < len(identifier)-1 {
			id += "|"
		}
	}

	config.Debugf("ResourceExists %v %v", typeName, id)

	_, err := getClient().GetResource(context.Background(), &cloudcontrol.GetResourceInput{
		Identifier: &id,
		TypeName:   &typeName,
	})

	if err != nil {
		config.Debugf("ResourceExists error: %v", err)
		return false
	}

	return true
}

// toJsonProps converts properties in a resource node to the JSON representation
func ToJsonProps(resource *yaml.Node) string {
	_, props := s11n.GetMapValue(resource, "Properties")
	if props == nil {
		return "{}"
	}
	p, _ := json.Marshal(format.Jsonise(props))
	return string(p)
}

func printProgress(p *types.ProgressEvent) string {
	status := ""
	if p.StatusMessage != nil {
		status = *p.StatusMessage
	}
	return fmt.Sprintf("ErrorCode: %v, Identifier: %v, OperationStatus: %v, ResourceModel: %v, StatusMessage: %v",
		p.ErrorCode, p.Identifier, p.OperationStatus, p.ResourceModel, status)
}

// CreateResource creates a resource based on the YAML node from the template,
// and blocks until resource creation is complete.
func CreateResource(logicalId string, resource *yaml.Node) (identifier string, model string, err error) {

	clientToken := uuid.New().String()

	// Intrinsics have already been resolved, so there should not
	// be any !Refs or !GetAtts, etc
	props := ToJsonProps(resource)

	config.Debugf("CreateResource props: %v", props)
	_, typeNode := s11n.GetMapValue(resource, "Type")
	if typeNode == nil {
		return identifier, model, fmt.Errorf("expected resource %v to have a Type", logicalId)
	}
	typeName := typeNode.Value
	input := cloudcontrol.CreateResourceInput{
		ClientToken:  &clientToken,
		DesiredState: &props,
		TypeName:     &typeName,
	}
	output, err := getClient().CreateResource(context.Background(), &input)

	config.Debugf("CreateResource output:\n%v", printProgress(output.ProgressEvent))

	if err != nil {
		return identifier, model, err
	}

	progress := output.ProgressEvent
	identifier, model, err = pollForCompletion(progress)
	if err != nil {
		return identifier, model, err
	}

	// Call GetResource to fill in the model, which for some reason is
	// not populated on the ProgressEvent above.
	model, err = GetResource(identifier, typeName)
	if err != nil {
		return identifier, model, err
	}

	return identifier, model, nil
}

// GetResource gets a resource from cloud control api
// It returns the resource model as a string
func GetResource(identifier string, typeName string) (string, error) {

	input := &cloudcontrol.GetResourceInput{
		Identifier: &identifier,
		TypeName:   &typeName,
	}

	result, err := getClient().GetResource(context.Background(), input)

	if err != nil {
		return "", err
	}

	return *result.ResourceDescription.Properties, nil

}

// pollForCompletion checks for progress until the operation is complete or fails
func pollForCompletion(progress *types.ProgressEvent) (string, string, error) {

	var identifier string
	var model string
	done := false

	// Poll for completion
	for !done {

		if progress.Identifier != nil {
			identifier = *progress.Identifier
		}
		if progress.ResourceModel != nil {
			model = *progress.ResourceModel
		}

		// config.Debugf("About to check OperationStatus, identifier: %v, model: %v", identifier, model)

		switch progress.OperationStatus {
		case "PENDING":
			done = false
		case "IN_PROGRESS":
			done = false
		case "SUCCESS":
			done = true
		case "FAILED":
			done = true
			msg := string(progress.ErrorCode)
			if progress.StatusMessage != nil {
				msg = *progress.StatusMessage
			}
			return identifier, model, fmt.Errorf("%v", msg)
		case "CANCEL_IN_PROGRESS":
			done = false
		case "CANCEL_COMPLETE":
			done = true
		default:
			config.Debugf("got unexpected status: %v", progress.OperationStatus)
		}

		if !done {
			status, statusErr := getClient().GetResourceRequestStatus(context.Background(),
				&cloudcontrol.GetResourceRequestStatusInput{
					RequestToken: progress.RequestToken,
				})
			if statusErr != nil {
				return identifier, model, statusErr // Is this terminal?
				// This is not a deployment failure. Network issue?
			}
			// config.Debugf("status:\n%v", printProgress(status.ProgressEvent))

			progress = status.ProgressEvent
		}
	}

	return identifier, model, nil
}

// Create a json patch document based on the new props and the prior json
// from the last version of the template (not the property model)
func CreatePatch(props *yaml.Node, priorJson string) (string, error) {

	config.Debugf("props: %v", node.ToSJson(props))

	jsonProps, err := json.Marshal(format.Jsonise(props))
	if err != nil {
		return "", err
	}

	// This is not right. The props might be a single required property,
	// and the priorModel will have everything, which then renders
	// a PatchDocument with a bunch of remove operations.
	// priorModel should be priorJson (from the old template)

	operations, err := jsonpatch.CreatePatch([]byte(priorJson), jsonProps)
	if err != nil {
		return "", err
	}

	// Sort the operations so tests are consistent
	sort.Slice(operations, func(i, j int) bool {
		return operations[i].Path < operations[j].Path
	})

	patchDocument := "[\n"
	first := true
	for _, operation := range operations {
		if first {
			first = false
		} else {
			patchDocument += ",\n"
		}
		patchDocument += fmt.Sprintf("    %s", operation.Json())
	}
	patchDocument += "\n]"

	config.Debugf("CreatePatch\n\njsonProps:\n%v\n\npriorJson:\n%v\n\nPatchDocument\nPatchDocument:\n%v",
		string(jsonProps), priorJson, string(patchDocument))

	return patchDocument, nil
}

// UpdateResource updates a resource based on the YAML node from the template,
// and blocks until resource update is complete.
func UpdateResource(
	logicalId string,
	identifier string,
	resource *yaml.Node,
	priorJson string) (model string, err error) {

	if logicalId == "" {
		return model, fmt.Errorf("logicalId is required for UpdateResource")
	}
	if identifier == "" {
		return model, fmt.Errorf("identifier is blank for UpdateResource %v", logicalId)
	}

	clientToken := uuid.New().String()

	_, typeNode := s11n.GetMapValue(resource, "Type")
	if typeNode == nil {
		return model, fmt.Errorf("expected resource %v to have a Type", logicalId)
	}
	typeName := typeNode.Value

	// Intrinsics have already been resolved, so there should not
	// be any !Refs or !GetAtts, etc

	_, props := s11n.GetMapValue(resource, "Properties")
	config.Debugf("UpdateResource %v props: %v", logicalId, node.ToSJson(props))

	// Create the patch document
	patchDocument, err := CreatePatch(props, priorJson)
	if err != nil {
		return model, nil
	}

	// AWS::SQS::Queue A: Failed: operation error CloudControl:
	// UpdateResource, https response error StatusCode: 400,
	// RequestID: 7773a459-504a-4562-a728-f0f2b6f9cd35,
	// api error ValidationException: Invalid patch update:
	// readOnlyProperties [/properties/QueueUrl, /properties/Arn] cannot be updated

	config.Debugf("PatchDocument for %v: %v", logicalId, patchDocument)
	input := cloudcontrol.UpdateResourceInput{
		ClientToken:   &clientToken,
		PatchDocument: &patchDocument,
		TypeName:      &typeName,
		Identifier:    &identifier,
	}
	output, err := getClient().UpdateResource(context.Background(), &input)
	if err != nil {
		return model, err
	}

	config.Debugf("UpdateResource output:\n%v", printProgress(output.ProgressEvent))

	progress := output.ProgressEvent
	_, model, err = pollForCompletion(progress)
	if err != nil {
		return model, err
	}

	// Call GetResource to fill in the model, which for some reason is
	// not populated on the ProgressEvent above.
	model, err = GetResource(identifier, typeName)
	if err != nil {
		return model, err
	}

	return model, nil
}

// DeleteResource deletes a resource and blocks until the operation is complete
func DeleteResource(logicalId string, identifier string, resource *yaml.Node) error {
	if logicalId == "" {
		return fmt.Errorf("logicalId is required for DeleteResource")
	}
	if identifier == "" {
		return fmt.Errorf("identifier is blank for DeleteResource %v", logicalId)
	}

	clientToken := uuid.New().String()

	_, typeNode := s11n.GetMapValue(resource, "Type")
	if typeNode == nil {
		return fmt.Errorf("expected resource %v to have a Type", logicalId)
	}
	typeName := typeNode.Value

	input := cloudcontrol.DeleteResourceInput{
		ClientToken: &clientToken,
		TypeName:    &typeName,
		Identifier:  &identifier,
	}
	output, err := getClient().DeleteResource(context.Background(), &input)

	if err != nil {
		return err
	}

	config.Debugf("DeleteResource output:\n%v", printProgress(output.ProgressEvent))

	progress := output.ProgressEvent
	_, _, err = pollForCompletion(progress)
	if err != nil {
		return err
	}

	return nil
}
