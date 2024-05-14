package forecast

import "github.com/aws-cloudformation/rain/internal/config"

// AWS::EC2::LaunchTemplate

func checkEC2LaunchTemplate(input PredictionInput) Forecast {

	forecast := makeForecast(input.typeName, input.logicalId)

	config.Debugf("About to check key name for launch template")

	// Check to see if the key name exists
	checkKeyName(&input, &forecast)

	// Make sure the AMI and the instance type are compatible
	checkInstanceType(&input, &forecast)

	return forecast

}
