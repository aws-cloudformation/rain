package ccapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
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

	done := false
	progress := output.ProgressEvent

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
			config.Debugf("CreateResource got unexpected status: %v", output.ProgressEvent.OperationStatus)
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
			config.Debugf("CreateResource status:\n%v", printProgress(status.ProgressEvent))

			progress = status.ProgressEvent
		}
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

// TODO - Update
// TODO - Delete
