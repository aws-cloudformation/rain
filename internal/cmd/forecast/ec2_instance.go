package forecast

import (
	"fmt"
	"github.com/aws-cloudformation/rain/internal/aws/ec2"
	"github.com/aws-cloudformation/rain/internal/aws/ssm"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
	"slices"
	"strings"
)

func checkKeyName(input *PredictionInput, forecast *Forecast) {

	var keyName string

	props := getPropNode(input)
	if props == nil {
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
			code := F0007
			if exists {
				forecast.Add(code, true, "Key exists")
			} else {
				forecast.Add(code, false, "Key does not exist")
			}

			spinner.Pop()
		} else {
			config.Debugf("%s.KeyName is empty", input.logicalId)
		}
	}
}

func resolveImageId(imageId string) string {
	// Strings in CloudFormation can look like this:
	// {{resolve:ssm:name}}
	// Make an API call to systems manager parameter store to get the value

	// TODO: This should probably move to where we resolve Refs

	// Check to see if the image id starts with {{resolve:ssm:
	if strings.HasPrefix(imageId, "{{resolve:ssm:") {
		// The image id is a parameter name
		imageId = strings.TrimPrefix(imageId, "{{resolve:ssm:")
		imageId = strings.TrimSuffix(imageId, "}}")

		config.Debugf("resolving %s", imageId)

		resolved, err := ssm.GetParameter(imageId)
		if err != nil {
			config.Debugf("failed to resolve %s: %v", imageId, err)
			return ""
		}
		config.Debugf("resolved %s to %s", imageId, resolved)
		return resolved
	}

	// TODO: Secrets manager

	return imageId
}

// checkInstanceType checks to see if the AMI and the instance type are compatible
func checkInstanceType(input *PredictionInput, forecast *Forecast) {

	var instanceType string
	code := F0008

	props := getPropNode(input)
	if props == nil {
		return
	}

	_, instanceTypeProp, _ := s11n.GetMapValue(props, "InstanceType")
	if instanceTypeProp == nil {
		config.Debugf("%s does not have InstanceType", input.logicalId)
		return
	}

	// If the name is a Ref, resolve it
	if instanceTypeProp.Kind == yaml.ScalarNode {
		// The name is hard coded
		instanceType = instanceTypeProp.Value
	} else {
		// We resolved Refs earlier so it should be a string
		config.Debugf("%s.InstanceType is not a string", input.logicalId)
		return
	}

	// Call the DescribeInstanceTypes API to get the instance type info
	spin(input.typeName, input.logicalId, "EC2 instance type exists?")
	instanceTypeInfo, err := ec2.GetInstanceType(instanceType)
	if err != nil {
		forecast.Add(code, false, err.Error())
	} else {
		forecast.Add(code, true, "Instance type exists")
	}
	spinner.Pop()

	// Make sure the instance type and AMI agree

	_, imageIdNode, _ := s11n.GetMapValue(props, "ImageId")
	if imageIdNode == nil {
		config.Debugf("%s does not have ImageId", input.logicalId)
		return
	}

	imageId := resolveImageId(imageIdNode.Value)

	spin(input.typeName, input.logicalId, "EC2 instance type matches AMI?")
	image, err := ec2.GetImage(imageId)
	if err != nil {
		forecast.Add(F0009, false, fmt.Sprintf("Image not found: %s", imageId))
		spinner.Pop()
		return
	}

	instanceTypesForArch, err := ec2.GetInstanceTypesForArchitecture(string(image.Architecture))
	if err != nil {
		config.Debugf("failed to get instance types for architecture %s: %v", image.Architecture, err)
		spinner.Pop()
		return
	}
	code = F0009
	if !slices.Contains(instanceTypesForArch, string(instanceTypeInfo.InstanceType)) {
		forecast.Add(code, false, fmt.Sprintf("Instance type %s does not support AMI %s", instanceType, imageId))
	} else {
		forecast.Add(code, true, "Instance type matches AMI")
	}
	spinner.Pop()
}

func getPropNode(input *PredictionInput) *yaml.Node {
	// Check to see if the resource has the InstanceType property set
	_, props, _ := s11n.GetMapValue(input.resource, "Properties")
	if props == nil {
		config.Debugf("expected %s to have Properties", input.logicalId)
		return nil
	}

	// If the input resource is an AWS::EC2::LaunchTemplate, props is LaunchTemplateData
	if input.typeName == "AWS::EC2::LaunchTemplate" {
		_, props, _ = s11n.GetMapValue(props, "LaunchTemplateData")
		if props == nil {
			config.Debugf("expected %s to have LaunchTemplateData", input.logicalId)
			return nil
		}
	}
	return props
}

func checkEC2Instance(input PredictionInput) Forecast {

	forecast := makeForecast(input.typeName, input.logicalId)

	// Check to see if the key name exists
	checkKeyName(&input, &forecast)

	// Make sure the AMI and the instance type are compatible
	checkInstanceType(&input, &forecast)

	return forecast

}
