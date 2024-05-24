package forecast

import (
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// checkLambdaFunction checks for potential stack failures related to functions
func checkLambdaFunction(input PredictionInput) Forecast {

	forecast := makeForecast(input.typeName, input.logicalId)

	_, props, _ := s11n.GetMapValue(input.resource, "Properties")
	if props == nil {
		config.Debugf("No Properties found for %s", input.logicalId)
		return forecast
	}
	_, roleProp, _ := s11n.GetMapValue(props, "Role")

	// If the role is specified, and it's a scalar, check if it exists
	if roleProp != nil && roleProp.Kind == yaml.ScalarNode {
		roleArn := roleProp.Value
		LineNumber = roleProp.Line
		if !iam.RoleExists(roleArn) {
			forecast.Add(F0016, false, "Role does not exist")
		} else {
			forecast.Add(F0016, true, "Role exists")
		}

		// Check to make sure the iam role can be assumed by the lambda function
		canAssume, err := iam.CanAssumeRole(roleArn, "lambda.amazonaws.com")
		if err != nil {
			config.Debugf("Error checking role: %s", err)
		} else {
			if !canAssume {
				forecast.Add(F0017, false, "Role can not be assumed")
			} else {
				forecast.Add(F0017, true, "Role can be assumed")
			}
		}
	}

	return forecast
}
