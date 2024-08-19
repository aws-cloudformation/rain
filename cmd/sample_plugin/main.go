package main

import (
	fc "github.com/aws-cloudformation/rain/plugins/forecast"
)

type PluginImpl struct{}

func predictLambda(input fc.PredictionInput) fc.Forecast {
	forecast := fc.MakeForecast(&input)

	forecast.Add("CODE", false, "testing plugin", 0)

	return forecast
}

func (p *PluginImpl) GetForecasters() map[string]func(input fc.PredictionInput) fc.Forecast {
	retval := make(map[string]func(input fc.PredictionInput) fc.Forecast)

	retval["AWS::Lambda::Function"] = predictLambda

	return retval
}

func main() {
}

var Plugin = PluginImpl{}
