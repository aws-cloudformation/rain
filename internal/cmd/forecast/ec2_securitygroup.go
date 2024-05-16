package forecast

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/aws/ec2"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

func checkEC2SecurityGroup(input PredictionInput) Forecast {

	// TODO - Cannot delete a security group that still has instances
	// (See if there are instances not in this stack)
	/* CLI commands

	aws ec2 describe-network-interfaces --filters Name=group-id,Values=<group-id> --region <region> --output json

	*/

	//	filters := []types.Filter{}

	//	input := *ec2.DescribeNetworkInterfacesInput{
	//		Filters:
	//	}

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

	// TODO: Security group is invalid without either a VpcId or a default VPC

	// TODO: Make sure security groups in a template are all in the same network
	// Accidentally leaving off the VpcId property means default, but others might be set

	forecast := makeForecast(input.typeName, input.logicalId)

	spin(input.typeName, input.logicalId, "Checking if all security groups are in the same VPC")
	checkSecurityGroupsInSameVPC(&input, &forecast)
	spinner.Pop()

	return forecast

}

func checkSecurityGroupsInSameVPC(input *PredictionInput, forecast *Forecast) {

	// Get the default VPC
	defaultVPCId, err := ec2.GetDefaultVPCId()
	if err != nil {
		msg := fmt.Sprintf("Unable to get default VPC Id: %v", err)
		forecast.Add(false, msg)
		return
	}

	config.Debugf("defaultVPCId: %s", defaultVPCId)

	resources, err := input.source.GetSection(cft.Resources)
	if err != nil {
		forecast.Add(false, fmt.Sprintf("%v", err))
		return
	}

	var vpcId string

	// Iterate through resources to find all security groups
	for i := 0; i < len(resources.Content); i += 2 {

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
		var rvpcid string
		if v != nil {
			config.Debugf("v: %s", node.ToSJson(v))
			if v.Kind == yaml.ScalarNode {
				rvpcid = v.Value
			} else if v.Kind == yaml.MappingNode {
				// Likely a Ref to the VPC created in this template
				if v.Content[0].Value == "Ref" {
					rvpcid = v.Content[1].Value
				}
			}
			if rvpcid == "" {
				// Not sure what this is
				config.Debugf("Unable to determine VpcId for %s", input.logicalId)
				return
			}
		} else {
			rvpcid = defaultVPCId
		}

		if vpcId == "" {
			vpcId = rvpcid
		} else if rvpcid != vpcId {
			msg := fmt.Sprintf("VPC ID for this security group (%s) does not match %s", rvpcid, vpcId)
			forecast.Add(false, msg)
			return
		}

	}
	forecast.Add(true, "All VPC Ids on security groups are the same")
}
