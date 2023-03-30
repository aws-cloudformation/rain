package forecast

func checkEC2Instance(input PredictionInput) Forecast {

	// TODO - Is this instance type available in this AZ?

	forecast := makeForecast(input.typeName, input.logicalId)

	return forecast

}
