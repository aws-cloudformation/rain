package forecast

import (
	"github.com/aws-cloudformation/rain/internal/aws/kms"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	fc "github.com/aws-cloudformation/rain/plugins/forecast"
)

func CheckSNSTopic(input fc.PredictionInput) fc.Forecast {

	forecast := makeForecast(input.TypeName, input.LogicalId)

	spin(input.TypeName, input.LogicalId, "Checking SNS Topic Key")
	checkSNSTopicKey(&input, &forecast)
	spinner.Pop()

	return forecast
}

func checkSNSTopicKey(input *fc.PredictionInput, forecast *fc.Forecast) {

	// Get the KmsMasterKeyId from the input resource properties
	k := input.GetPropertyNode("KmsMasterKeyId")
	if k != nil {
		keyArn := k.Value
		valid := kms.IsKeyArnValid(keyArn)
		if valid {
			forecast.Add(F0013, true, "KMS Key is valid", input.Resource.Line)
		} else {
			forecast.Add(F0013, false, "KMS Key is invalid", input.Resource.Line)
		}
	}
}
