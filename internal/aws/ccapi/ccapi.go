package ccapi

import (
	"context"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudcontrol"
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
