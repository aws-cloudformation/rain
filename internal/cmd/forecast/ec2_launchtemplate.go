package forecast

// AWS::EC2::LaunchTemplate

func CheckEC2LaunchTemplate(input PredictionInput) Forecast {

	forecast := makeForecast(input.typeName, input.logicalId)

	// Check to see if the key name exists
	checkKeyName(&input, &forecast)

	// Make sure the AMI and the instance type are compatible
	checkInstanceType(&input, &forecast)

	return forecast

}
