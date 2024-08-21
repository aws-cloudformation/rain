package main

import (
	fc "github.com/aws-cloudformation/rain/plugins/forecast"
)

type PluginImpl struct{}

// A sample prediction function.
func predictLambda(input fc.PredictionInput) fc.Forecast {
	forecast := fc.MakeForecast(&input)

	// Implement whatever checks you want to make here
	forecast.Add("CODE", false, "testing plugin", 0)

	return forecast
}

// GetForecasters must be implemented by forecast plugins
// This function returns all predictions functions implemented by the plugin
func (p *PluginImpl) GetForecasters() map[string]func(input fc.PredictionInput) fc.Forecast {
	retval := make(map[string]func(input fc.PredictionInput) fc.Forecast)

	retval["AWS::Lambda::Function"] = predictLambda

	return retval
}

// Leave main empty
func main() {
}

// This variable is required to allow rain to find the implementation
var Plugin = PluginImpl{}
