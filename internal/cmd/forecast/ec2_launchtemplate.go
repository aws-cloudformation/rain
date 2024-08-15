package forecast

import fc "github.com/aws-cloudformation/rain/plugins/forecast"

// AWS::EC2::LaunchTemplate

func CheckEC2LaunchTemplate(input fc.PredictionInput) fc.Forecast {

	forecast := makeForecast(input.TypeName, input.LogicalId)

	// Check to see if the key name exists
	checkKeyName(&input, &forecast)

	// Make sure the AMI and the instance type are compatible
	checkInstanceType(&input, &forecast)

	return forecast

}
