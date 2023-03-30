package forecast

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

	forecast := makeForecast(input.typeName, input.logicalId)

	return forecast

}
