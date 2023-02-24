package forecast

import (
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
)

// Returns true if the user has the required permissions on the resource
// verb is create, delete, or update
func checkPermissions(input PredictionInput, resourceArn string, verb string) (bool, []string) {

	spin(input.typeName, input.logicalId, "permitted?")

	// Go get the list of permissions from the registry
	actions, err := cfn.GetTypePermissions("AWS::S3::Bucket", verb)
	if err != nil {
		return false, []string{err.Error()}
	}

	// Update the spinner with the action being checked
	spinnerCallback := func(action string) {
		spin(input.typeName, input.logicalId, action+" permitted?")
	}

	// Simulate the actions
	result, messages := iam.Simulate(actions, resourceArn, RoleArn, spinnerCallback)

	spinner.Pop()
	return result, messages
}
