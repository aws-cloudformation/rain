package forecast

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"golang.org/x/exp/slices"
)

// Returns true if the user has the required permissions on the resource
// verb is create, delete, or update
func checkTypePermissions(input PredictionInput, resourceArn string, verb string) (bool, []string) {

	spin(input.typeName, input.logicalId, "permitted?")

	// Go get the list of permissions from the registry
	actions, err := cfn.GetTypePermissions(input.typeName, verb)
	if err != nil {
		return false, []string{err.Error()}
	}

	// Not all of these will work with the Simulator, since the registry
	// schema also includes permissions for related services.
	// For example, the create permissions on a lambda function include s3:GetObject,
	// and we don't know what the arn would be.
	// This means we are not checking everything that could go wrong.
	// TODO - Is there a way we can figure out the arns for related services?
	// This would likely not be practical in a generic way,
	// but it's something we should eventually add to custom handling for each service.
	svcLower := strings.ToLower(strings.Split(input.typeName, "::")[1])
	actionsToRemove := make([]string, 0)
	for _, action := range actions {
		// Remove all actions that don't belong to this service
		if !strings.HasPrefix(action, svcLower) {
			actionsToRemove = append(actionsToRemove, action)
		}
	}

	// Exceptions
	// Sometimes the registry documents actions that the resource type
	// might need in some situations, but these checks will fail under
	// other circumstances, and it's not easy to know when they are relevant.
	if svcLower == "lambda" {
		actionsToRemove = append(actionsToRemove, "lambda:GetCodeSigningConfig")
		actionsToRemove = append(actionsToRemove, "lambda:GetLayerVersion")
	}
	if input.typeName == "AWS::IAM::Policy" {
		actionsToRemove = append(actionsToRemove, "iam:PutUserPolicy")
		actionsToRemove = append(actionsToRemove, "iam:PutRolePolicy")
		actionsToRemove = append(actionsToRemove, "iam:PutGroupPolicy")
		actionsToRemove = append(actionsToRemove, "iam:DeleteRolePolicy")
		actionsToRemove = append(actionsToRemove, "iam:DeleteUserPolicy")
		actionsToRemove = append(actionsToRemove, "iam:DeleteGroupPolicy")
		actionsToRemove = append(actionsToRemove, "iam:GetRolePolicy")
		actionsToRemove = append(actionsToRemove, "iam:GetUserPolicy")
		actionsToRemove = append(actionsToRemove, "iam:GetGroupPolicy")
	}

	// Make a new slice with the actions we care about
	actionsToCheck := make([]string, 0)
	for _, action := range actions {
		if !slices.Contains(actionsToRemove, action) {
			actionsToCheck = append(actionsToCheck, action)
		}
	}

	// Update the spinner with the action being checked
	spinnerCallback := func(action string) {
		spin(input.typeName, input.logicalId, action+" permitted?")
	}

	// Simulate the actions
	result, messages := iam.Simulate(actionsToCheck,
		resourceArn, input.roleArn, spinnerCallback)

	spinner.Pop()
	return result, messages
}

// Check permissions to make sure the current role can create-update-delete
func checkPermissions(input PredictionInput, forecast *Forecast) error {
	resourceArn := predictResourceArn(input)
	if resourceArn == "" {
		// We don't know how to make an arn for this type
		config.Debugf("Can't check permissions for %v %v, ARN unknown", input.typeName, input.logicalId)
		return nil
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
	return nil
}
