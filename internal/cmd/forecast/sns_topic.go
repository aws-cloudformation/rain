package forecast

import (
	"github.com/aws-cloudformation/rain/internal/aws/kms"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
)

func CheckSNSTopic(input PredictionInput) Forecast {

	forecast := makeForecast(input.typeName, input.logicalId)

	spin(input.typeName, input.logicalId, "Checking SNS Topic Key")
	checkSNSTopicKey(&input, &forecast)
	spinner.Pop()

	return forecast
}

func checkSNSTopicKey(input *PredictionInput, forecast *Forecast) {

	// Get the KmsMasterKeyId from the input resource properties
	k := input.GetPropertyNode("KmsMasterKeyId")
	if k != nil {
		keyArn := k.Value
		valid := kms.IsKeyArnValid(keyArn)
		if valid {
			forecast.Add(F0013, true, "KMS Key is valid")
		} else {
			forecast.Add(F0013, false, "KMS Key is invalid")
		}
	}
}
