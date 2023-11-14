package ccapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

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
func toJsonProps(resource *yaml.Node) string {
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
	props := toJsonProps(resource)

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

		config.Debugf("About to check OperationStatus, identifier: %v, model: %v", identifier, model)

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
			config.Debugf("status:\n%v", printProgress(status.ProgressEvent))

			progress = status.ProgressEvent
		}
	}

	return identifier, model, nil
}

// patch is used to construct a PatchDocument
type patch struct {
	// add, remove, replace, move, copy, and test
	Op string `json:"op"`

	// For example, /PropertyName
	Path string `json:"path"`

	// Ints and Bools have to be converted from strings to match the schema
	Value any `json:"value"`
}

// yamlVal converts node.Value to a string, int, or bool based on the Tag
func yamlVal(n *yaml.Node) (any, error) {
	var v any
	var err error
	switch n.Tag {
	case "!!bool":
		v, err = strconv.ParseBool(n.Value)
		if err != nil {
			return "", err
		}
	case "!!int":
		v, err = strconv.ParseInt(n.Value, 10, 32)
		if err != nil {
			return "", err
		}
	default:
		v = string(n.Value)
	}
	return v, nil
}

// patchPath is a recursive function that builds the PatchDocument
func patchPath(patches []patch, props *yaml.Node, path string) ([]patch, error) {

	var err error
	for i, p := range props.Content {
		if i%2 == 0 {
			name := p.Value
			newPath := path + "/" + name
			val := props.Content[i+1]
			if val.Kind == yaml.ScalarNode {
				v, err := yamlVal(val)
				if err != nil {
					return patches, err
				}
				patches = append(patches, patch{Op: "replace", Path: newPath, Value: v})
			} else if val.Kind == yaml.SequenceNode {
				// TODO
			} else if val.Kind == yaml.MappingNode {
				// Recurse
				patches, err = patchPath(patches, val, newPath)
				if err != nil {
					return patches, err
				}
			}
		}
	}

	return patches, nil
}

// createPatch converts the template properties into a JSON Patch Document
func createPatch(props *yaml.Node) (string, error) {

	/*
		We have to create a PatchDocument here with json.
		op: add, remove, replace, move, copy, and test

		[
		  {
			"op": "test",
			"path": "/RetentionInDays",
			"value":3653
		  },
		  {
			"op": "replace",
			"path": "/RetentionInDays",
			"value":180
		  }
		]
	*/
	patches := make([]patch, 0)

	// Iterate through all changes to the properties on the resource
	// A:1 becomes path: /A, value: 1
	// A:
	//   B:
	//     C: 1
	// becomes path: /A/B/C, value 1

	config.Debugf("patchPath: %v", node.ToJson(props))

	var err error
	patches, err = patchPath(patches, props, "")
	if err != nil {
		return "", err
	}

	config.Debugf("patches: %v", patches)

	m, err := json.Marshal(patches)
	if err != nil {
		return "", err
	}
	p := string(m)
	return p, nil
}

// UpdateResource updates a resource based on the YAML node from the template,
// and blocks until resource update is complete.
func UpdateResource(
	logicalId string,
	identifier string,
	resource *yaml.Node,
	priorModel string) (model string, err error) {

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

	// TODO - To create a patch document correctly, we need to compare the props
	// to the prior model.

	patchDocument, err := createPatch(props)
	if err != nil {
		return model, nil
	}

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
