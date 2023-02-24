package forecast

import (
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
)

// Returns true if the user has the required permissions on the resource
// verb is create, delete, or update
func checkPermissions(input PredictionInput, resourceArn string, verb string) (bool, string) {

	spin(input.typeName, input.logicalId, "permitted?")

	// Go get the list of permissions from the registry
	actions, err := cfn.GetTypePermissions("AWS::S3::Bucket", verb)
	if err != nil {
		return false, err.Error()
	}
	result, err := iam.Simulate(actions, resourceArn, RoleArn)
	if err != nil {
		return false, err.Error()
	}
	if !result {
		return false, "Insufficient permissions"
	}

	spinner.Pop()

	return true, ""
}
