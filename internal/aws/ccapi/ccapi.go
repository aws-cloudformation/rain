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

// CreateResource creates a resource based on the YAML node from the template,
// and blocks until resource creation is complete.
func CreateResource(logicalId string, resource *yaml.Node) error {
	clientToken := uuid.New().String()
	props := toJsonProps(resource)
	config.Debugf("CreateResource props: %v", props)
	_, typeNode := s11n.GetMapValue(resource, "Type")
	if typeNode == nil {
		return fmt.Errorf("expected resource to have a Type", logicalId)
	}
	typeName := typeNode.Value
	input := cloudcontrol.CreateResourceInput{
		ClientToken:  &clientToken,
		DesiredState: &props,
		TypeName:     &typeName,
	}
	output, err := getClient().CreateResource(context.Background(), &input)

	config.Debugf("CreateResource output:\n%v", output)

	if err != nil {
		return err
	}

	done := false
	progress := output.ProgressEvent

	// Poll for completion
	for !done {

		switch progress.OperationStatus {
		case "PENDING":
			done = false
		case "IN_PROGRESS":
			done = false
		case "SUCCESS":
			done = true
		case "FAILED":
			done = true
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
				return statusErr // Is this terminal?
			}
			config.Debugf("CreateResource status:\n%v", status)
			progress = status.ProgressEvent
		}
	}

	return nil
}

// TODO - Update
// TODO - Delete
