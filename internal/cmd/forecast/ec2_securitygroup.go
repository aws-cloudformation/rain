package forecast

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/aws/ec2"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/s11n"
	fc "github.com/aws-cloudformation/rain/plugins/forecast"
	"gopkg.in/yaml.v3"
)

func CheckEC2SecurityGroup(input fc.PredictionInput) fc.Forecast {

	// TODO - Cannot delete a security group that still has instances
	// (See if there are instances not in this stack)

	// TODO - Make sure security group exists in VPC
	/*
		Security group does not exist in VPC

		Verify that the security group exists in the VPC that you specified. If the
		security group exists, ensure that you specify the security group ID and
		not the security group name. For example, the
		AWS::EC2::SecurityGroupIngress resource has a SourceSecurityGroupName and
		SourceSecurityGroupId properties. For VPC security groups, you must use the
		SourceSecurityGroupId property and specify the security group ID.
	*/

	forecast := makeForecast(input.TypeName, input.LogicalId)

	spin(input.TypeName, input.LogicalId, "Checking if all security groups are in the same VPC")
	checkSecurityGroupsInSameVPC(&input, &forecast)
	spinner.Pop()

	return forecast

}

// Not sure if we can accurately check if the security group is in use,
// since the instances may come from this template and will be deleted first.
// We could cross-check the physical IDs if it's actually an instance,
// but more likely will be instances from an ASG and how do we correlate
// them with the security group? Likely to have false positives.

//// checkSecurityGroupIsInUse checks on deletes if a security group cannot be deleted
//func checkSecurityGroupIsInUse(input *PredictionInput, forecast *Forecast) {
//	if input.stackExists && action == DELETE {
//		// Check if the security group is in use
//		sgInUse, err := ec2.IsSecurityGroupInUse(input.stackName)
//		if err != nil {
//			forecast.Add(false, fmt.Sprintf("%v", err))
//			return
//		}
//		if sgInUse {
//			msg := fmt.Sprintf("Security group is in use by other resources")
//			forecast.Add(false, msg)
//			return
//		}
//		forecast.Add(true, "Security group is not in use by other resources")
//	}
//}

func checkSecurityGroupsInSameVPC(input *fc.PredictionInput, forecast *fc.Forecast) {

	code := F0010

	// Get the default VPC
	defaultVPCId, err := ec2.GetDefaultVPCId()
	if err != nil {
		msg := fmt.Sprintf("Unable to get default VPC Id: %v", err)
		forecast.Add(code, false, msg, input.Resource.Line)
		return
	}

	config.Debugf("defaultVPCId: %s", defaultVPCId)

	resources, err := input.Source.GetSection(cft.Resources)
	if err != nil {
		config.Debugf("Unable to get Resources: %v", err)
		return
	}

	var vpcId string

	// Iterate through resources to find all security groups
	for i := 0; i < len(resources.Content); i += 2 {

		thisLogicalId := resources.Content[i].Value

		_, typ, _ := s11n.GetMapValue(resources.Content[i+1], "Type")
		if typ == nil {
			config.Debugf("expected %s to have Type", resources.Content[i].Value)
			continue
		}
		if typ.Value != "AWS::EC2::SecurityGroup" {
			continue
		}
		_, props, _ := s11n.GetMapValue(resources.Content[i+1], "Properties")
		if props == nil {
			config.Debugf("expected %s to have Properties", resources.Content[i].Value)
			continue
		}

		// See if VpcId is set. If not, assume it's the default.
		// If it is hard coded or set in a param, make sure they are all the same
		_, v, _ := s11n.GetMapValue(props, "VpcId")
		var resourceVpcId string
		if v != nil {
			if v.Kind == yaml.ScalarNode {
				resourceVpcId = v.Value
			} else if v.Kind == yaml.MappingNode {
				// Likely a Ref to the VPC created in this template
				if v.Content[0].Value == "Ref" {
					resourceVpcId = v.Content[1].Value
				}
			}
			if resourceVpcId == "" {
				// Not sure what this is
				config.Debugf("Unable to determine VpcId for %s", input.LogicalId)
				return
			}
		} else {
			resourceVpcId = defaultVPCId
			if defaultVPCId == "" {
				msg := fmt.Sprintf("There is no default VPC and VpcId is not set on %s", thisLogicalId)
				forecast.Add(F0011, false, msg, input.Resource.Line)
			}
		}

		if vpcId == "" {
			vpcId = resourceVpcId
		} else if resourceVpcId != vpcId {
			msg := fmt.Sprintf("VPC ID for this security group (%s) does not match %s", resourceVpcId, vpcId)
			forecast.Add(code, false, msg, input.Resource.Line)
			return
		}

	}
	forecast.Add(code, true, "All VPC Ids on security groups are the same",
		input.Resource.Line)
}
