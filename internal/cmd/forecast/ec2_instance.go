package forecast

import (
	"github.com/aws-cloudformation/rain/internal/aws/ec2"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

func checkKeyName(input *PredictionInput, forecast *Forecast) {

	var keyName string

	// Check to see if the resource has the KeyName property set
	_, props, _ := s11n.GetMapValue(input.resource, "Properties")
	if props == nil {
		config.Debugf("expected %s to have Properties", input.logicalId)
		return
	}

	_, keyNameProp, _ := s11n.GetMapValue(props, "KeyName")
	if keyNameProp != nil {

		// If the name is a Ref, resolve it
		if keyNameProp.Kind == yaml.ScalarNode {
			// The name is hard coded
			keyName = keyNameProp.Value
		} else {
			// We resolved Refs earlier so it should be a string
			config.Debugf("%s.KeyName is not a string", input.logicalId)
			return
		}

		if keyName != "" {

			// Check to see if the key exists
			spin(input.typeName, input.logicalId, "EC2 instance key exists?")

			exists, _ := ec2.CheckKeyPairExists(keyName)
			if exists {
				forecast.Add(true, "Key exists")
			} else {
				forecast.Add(false, "Key does not exist")
			}

			spinner.Pop()
		} else {
			config.Debugf("%s.KeyName is empty", input.logicalId)
		}
	}
}

func checkEC2Instance(input PredictionInput) Forecast {

	// TODO - Is this instance type available in this AZ?

	forecast := makeForecast(input.typeName, input.logicalId)

	// Check to see if the key name exists
	checkKeyName(&input, &forecast)

	return forecast

}
