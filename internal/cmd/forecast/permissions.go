package forecast

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/google/uuid"
)

// Returns an arn that matches what the resource arn will be
// with the given resource name (physical id).
// Returns "" if we don't know how to make the arn
func predictResourceArn(input PredictionInput, resourceName string) string {
	switch input.typeName {
	case "AWS::S3::Bucket":
		return fmt.Sprintf("arn:aws:s3:::%v", resourceName)
	default:
		return ""
	}
}

// Returns true if the user has the required permissions on the resource
// verb is create, delete, or update
func checkTypePermissions(input PredictionInput, resourceArn string, verb string) (bool, []string) {

	spin(input.typeName, input.logicalId, "permitted?")

	// Go get the list of permissions from the registry
	actions, err := cfn.GetTypePermissions(input.typeName, verb)
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

// Check permissions to make sure the current role can create-update-delete
func checkPermissions(input PredictionInput, forecast *Forecast) {
	// Make up a resource name if it doesn't exist yet
	resourceName := fmt.Sprintf("rain-%v", uuid.New())
	if input.stackExists {
		res, err := cfn.GetStackResource(input.stackName, input.logicalId)
		if err != nil {
			// The resource exists
			resourceName = *res.PhysicalResourceId
		}
	}
	resourceArn := predictResourceArn(input, resourceName)
	if resourceArn == "" {
		// We don't know how to make an arn for this type
		config.Debugf("Can't check permissions for %v %v, ARN unknown", input.typeName, input.logicalId)
		return
	}

	var ok bool
	var reason []string
	if input.stackExists {
		ok, reason = checkTypePermissions(input, resourceArn, "update")
		if !ok {
			forecast.Add(false,
				fmt.Sprintf("Insufficient permissions to update %v\n\t%v", resourceArn, strings.Join(reason, "\n\t")))
		} else {
			forecast.Add(true, "Role has update permissions")
		}

		ok, reason = checkTypePermissions(input, resourceArn, "delete")
		if !ok {
			forecast.Add(false,
				fmt.Sprintf("Insufficient permissions to delete %v\n\t%v", resourceArn, strings.Join(reason, "\n\t")))
		} else {
			forecast.Add(true, "Role has delete permissions")
		}
	} else {
		ok, reason = checkTypePermissions(input, resourceArn, "create")
		if !ok {
			forecast.Add(false,
				fmt.Sprintf("Insufficient permissions to create %v\n\t%v", resourceArn, strings.Join(reason, "\n\t")))
		} else {
			forecast.Add(true, "Role has create permissions")
		}
	}
}
