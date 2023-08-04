package forecast

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/google/uuid"
)

// Returns an arn that matches the format for the resource.
// Returns "" if we don't know how to make the arn, we haven't implemented it yet, or if
// the resource does not have Arns that we can use to plug in to the simulator.
// Supported services:
// S3
// ...
func predictResourceArn(input PredictionInput) string {

	// There's not a great way to do this in a truly generic fashion.
	// There is no direct map between IAM resource ARNs and CloudFormation resource types.

	// We considered trying to generate something based on the arn patterns in
	// https://awspolicygen.s3.amazonaws.com/js/policies.js

	// Make up an id if it doesn't exist yet
	physicalId := fmt.Sprintf("rain-%v", uuid.New())
	if input.stackExists {
		res, err := cfn.GetStackResource(input.stackName, input.logicalId)
		if err != nil {
			// The resource exists
			physicalId = *res.PhysicalResourceId
		}
	}

	// There might be restrictions on the format of a physical id, so
	// we might have to change it per resource type.

	// Some resources have complex arns that will require us to inspect input.resource

	// To add new resources here, a good way to start is by opening both the CloudFormation
	// docs for the service, and the Service Authorization Reference (Resource Types section)
	//
	// for example
	//
	// https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/AWS_S3.html
	// and
	// https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazons3.html#amazons3-resources-for-iam-policies

	switch input.typeName {
	case "AWS::S3::Bucket":
		return fmt.Sprintf("arn:aws:s3:::%v", physicalId)
	case "AWS::S3::BucketPolicy":
		return ""
	case "AWS::S3::AccessPoint":
		// arn:${Partition}:s3:${Region}:${Account}:accesspoint/${AccessPointName}
		return fmt.Sprintf("arn:%v:s3:%v:%v:accesspoint/%v",
			input.env.partition, input.env.region, input.env.account, physicalId)
	case "AWS::S3::MultiRegionAccessPoint":
		// arn:${Partition}:s3::${Account}:accesspoint/${AccessPointAlias}
		return fmt.Sprintf("arn:%v:s3::%v:accesspoint/%v",
			input.env.partition, input.env.account, physicalId)
	case "AWS::S3::MultiRegionAccessPointPolicy":
		return ""
	case "AWS::S3::StorageLens":
		// arn:${Partition}:s3:${Region}:${Account}:storage-lens/${ConfigId}
		return fmt.Sprintf("arn:%v:s3:%v:%v:storage-lens/%v",
			input.env.partition, input.env.region, input.env.account, physicalId)
	case "AWS::EC2::CapacityReservation":
		return ""
	case "AWS::EC2::CapacityReservationFleet":
		return ""
	case "AWS::EC2::CarrierGateway":
		return ""
	case "AWS::EC2::ClientVpnAuthorizationRule":
		return ""
	case "AWS::EC2::ClientVpnEndpoint":
		return ""
	case "AWS::EC2::ClientVpnRoute":
		return ""
	case "AWS::EC2::ClientVpnTargetNetworkAssociation":
		return ""
	case "AWS::EC2::CustomerGateway":
		return ""
	case "AWS::EC2::DHCPOptions":
		return ""
	case "AWS::EC2::EC2Fleet":
		return ""
	case "AWS::EC2::EgressOnlyInternetGateway":
		return ""
	case "AWS::EC2::EIP":
		return ""
	case "AWS::EC2::EIPAssociation":
		return ""
	case "AWS::EC2::EnclaveCertificateIamRoleAssociation":
		return ""
	case "AWS::EC2::FlowLog":
		return ""
	case "AWS::EC2::GatewayRouteTableAssociation":
		return ""
	case "AWS::EC2::Host":
		return ""
	case "AWS::EC2::Instance":
		return ""
	case "AWS::EC2::InternetGateway":
		return ""
	case "AWS::EC2::IPAM":
		return ""
	case "AWS::EC2::IPAMAllocation":
		return ""
	case "AWS::EC2::IPAMPool":
		return ""
	case "AWS::EC2::IPAMPoolCidr":
		return ""
	case "AWS::EC2::IPAMResourceDiscovery":
		return ""
	case "AWS::EC2::IPAMResourceDiscoveryAssociation":
		return ""
	case "AWS::EC2::IPAMScope":
		return ""
	case "AWS::EC2::KeyPair":
		return ""
	case "AWS::EC2::LaunchTemplate":
		return ""
	case "AWS::EC2::LocalGatewayRoute":
		return ""
	case "AWS::EC2::LocalGatewayRouteTable":
		return ""
	case "AWS::EC2::LocalGatewayRouteTableVirtualInterfaceGroupAssociation":
		return ""
	case "AWS::EC2::LocalGatewayRouteTableVPCAssociation":
		return ""
	case "AWS::EC2::NatGateway":
		return ""
	case "AWS::EC2::NetworkAcl":
		return ""
	case "AWS::EC2::NetworkAclEntry":
		return ""
	case "AWS::EC2::NetworkInsightsAccessScope":
		return ""
	case "AWS::EC2::NetworkInsightsAccessScopeAnalysis":
		return ""
	case "AWS::EC2::NetworkInsightsAnalysis":
		return ""
	case "AWS::EC2::NetworkInsightsPath":
		return ""
	case "AWS::EC2::NetworkInterface":
		return ""
	case "AWS::EC2::NetworkInterfaceAttachment":
		return ""
	case "AWS::EC2::NetworkInterfacePermission":
		return ""
	case "AWS::EC2::NetworkPerformanceMetricSubscription":
		return ""
	case "AWS::EC2::PlacementGroup":
		return ""
	case "AWS::EC2::PrefixList":
		return ""
	case "AWS::EC2::Route":
		return ""
	case "AWS::EC2::RouteTable":
		return ""
	case "AWS::EC2::SecurityGroup":
		return ""
	case "AWS::EC2::SecurityGroupEgress":
		return ""
	case "AWS::EC2::SecurityGroupIngress":
		return ""
	case "AWS::EC2::SpotFleet":
		return ""
	case "AWS::EC2::Subnet":
		return ""
	case "AWS::EC2::SubnetCidrBlock":
		return ""
	case "AWS::EC2::SubnetNetworkAclAssociation":
		return ""
	case "AWS::EC2::SubnetRouteTableAssociation":
		return ""
	case "AWS::EC2::TrafficMirrorFilter":
		return ""
	case "AWS::EC2::TrafficMirrorFilterRule":
		return ""
	case "AWS::EC2::TrafficMirrorSession":
		return ""
	case "AWS::EC2::TrafficMirrorTarget":
		return ""
	case "AWS::EC2::TransitGateway":
		return ""
	case "AWS::EC2::TransitGatewayAttachment":
		return ""
	case "AWS::EC2::TransitGatewayConnect":
		return ""
	case "AWS::EC2::TransitGatewayMulticastDomain":
		return ""
	case "AWS::EC2::TransitGatewayMulticastDomainAssociation":
		return ""
	case "AWS::EC2::TransitGatewayMulticastGroupMember":
		return ""
	case "AWS::EC2::TransitGatewayMulticastGroupSource":
		return ""
	case "AWS::EC2::TransitGatewayPeeringAttachment":
		return ""
	case "AWS::EC2::TransitGatewayRoute":
		return ""
	case "AWS::EC2::TransitGatewayRouteTable":
		return ""
	case "AWS::EC2::TransitGatewayRouteTableAssociation":
		return ""
	case "AWS::EC2::TransitGatewayRouteTablePropagation":
		return ""
	case "AWS::EC2::TransitGatewayVpcAttachment":
		return ""
	case "AWS::EC2::VerifiedAccessEndpoint":
		return ""
	case "AWS::EC2::VerifiedAccessGroup":
		return ""
	case "AWS::EC2::VerifiedAccessInstance":
		return ""
	case "AWS::EC2::VerifiedAccessTrustProvider":
		return ""
	case "AWS::EC2::Volume":
		return ""
	case "AWS::EC2::VolumeAttachment":
		return ""
	case "AWS::EC2::VPC":
		return ""
	case "AWS::EC2::VPCCidrBlock":
		return ""
	case "AWS::EC2::VPCDHCPOptionsAssociation":
		return ""
	case "AWS::EC2::VPCEndpoint":
		return ""
	case "AWS::EC2::VPCEndpointConnectionNotification":
		return ""
	case "AWS::EC2::VPCEndpointService":
		return ""
	case "AWS::EC2::VPCEndpointServicePermissions":
		return ""
	case "AWS::EC2::VPCGatewayAttachment":
		return ""
	case "AWS::EC2::VPCPeeringConnection":
		return ""
	case "AWS::EC2::VPNConnection":
		return ""
	case "AWS::EC2::VPNConnectionRoute":
		return ""
	case "AWS::EC2::VPNGateway":
		return ""
	case "AWS::EC2::VPNGatewayRoutePropagation":
	default:
		return ""
	}
	return ""
}

// Returns true if the user has the required permissions on the resource
// verb is create, delete, or update
func checkTypePermissions(input PredictionInput, resourceArn string, verb string) (bool, []string) {

	spin(input.typeName, input.logicalId, "permitted?")

	// Go get the list of permissions from the registry
	actions, err := cfn.GetTypePermissions(input.typeName, verb)
	if err != nil {
		return false, []string{err.Error()}
	}

	// Update the spinner with the action being checked
	spinnerCallback := func(action string) {
		spin(input.typeName, input.logicalId, action+" permitted?")
	}

	// Simulate the actions
	result, messages := iam.Simulate(actions, resourceArn, input.roleArn, spinnerCallback)

	spinner.Pop()
	return result, messages
}

// Check permissions to make sure the current role can create-update-delete
func checkPermissions(input PredictionInput, forecast *Forecast) error {
	resourceArn := predictResourceArn(input)
	if resourceArn == "" {
		// We don't know how to make an arn for this type
		config.Debugf("Can't check permissions for %v %v, ARN unknown", input.typeName, input.logicalId)
		return nil
	}

	var ok bool
	var reason []string
	if input.stackExists {
		ok, reason = checkTypePermissions(input, resourceArn, "update")
		if !ok {
			forecast.Add(false,
				fmt.Sprintf("Insufficient permissions to update %v\n\t%v", resourceArn, strings.Join(reason, "\n\t")))
		} else {
			forecast.Add(true, "Role has update permissions")
		}

		ok, reason = checkTypePermissions(input, resourceArn, "delete")
		if !ok {
			forecast.Add(false,
				fmt.Sprintf("Insufficient permissions to delete %v\n\t%v", resourceArn, strings.Join(reason, "\n\t")))
		} else {
			forecast.Add(true, "Role has delete permissions")
		}
	} else {
		ok, reason = checkTypePermissions(input, resourceArn, "create")
		if !ok {
			forecast.Add(false,
				fmt.Sprintf("Insufficient permissions to create %v\n\t%v", resourceArn, strings.Join(reason, "\n\t")))
		} else {
			forecast.Add(true, "Role has create permissions")
		}
	}
	return nil
}
