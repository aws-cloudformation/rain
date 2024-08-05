package forecast

import "github.com/aws-cloudformation/rain/internal/config"

// AWS::AutoScaling::LaunchConfiguration

func CheckAutoScalingLaunchConfiguration(input PredictionInput) Forecast {

	forecast := makeForecast(input.typeName, input.logicalId)

	config.Debugf("About to check key name for launch config")

	// Check to see if the key name exists
	checkKeyName(&input, &forecast)

	return forecast

}
