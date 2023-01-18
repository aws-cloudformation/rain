package forecast

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/config"
)

// Returns true if the user has the required permissions on the resource
// verb is create, delete, or update
func checkPermissions(input PredictionInput, resourceArn string, verb string) bool {

	config.Debugf("Checking for permissions on %v", resourceArn)

	allAllowed := true

	// Go get the list of permissions from the registry
	actions, err := cfn.GetTypePermissions("AWS::S3::Bucket", verb)
	if err != nil {
		fmt.Println("Unable to get type permissions", err)
		return false
	}
	result, err := iam.Simulate(actions, resourceArn, Role)
	if err != nil {
		return false
	}
	if !result {
		allAllowed = false
	}

	return allAllowed
}
