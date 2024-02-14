package forecast

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/google/uuid"
)

// predictResourceArn returns an arn that matches the format for the resource.
// Returns "" if we don't know how to make the arn, we haven't implemented it yet, or if
// the resource does not have Arns that we can use to plug in to the simulator.
// Supported services: S3, EC2
func predictResourceArn(input PredictionInput) string {

	// There's not a great way to do this in a truly generic fashion.
	// There is no direct map between IAM resource ARNs and CloudFormation resource types.

	// We considered trying to generate something based on the arn patterns in
	// https://awspolicygen.s3.amazonaws.com/js/policies.js

	// Make up an id if it doesn't exist yet
	physicalId := fmt.Sprintf("rain-%v", uuid.New())

	if input.stackExists {

		config.Debugf("predictResourceArn stack exists")

		res, err := cfn.GetStackResource(input.stackName, input.logicalId)
		if err == nil {
			// The resource exists
			physicalId = *res.PhysicalResourceId

			// This physical id is not super useful
			// It's often an internal id that is not part of the arn

			config.Debugf("predictResourceArn physicalId: %v", physicalId)
		} else {
			config.Debugf("predictResourceArn got an error trying to get stack resource: %v", err)
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
		return standardFormat("s3", "storage-lens", input, physicalId)
	case "AWS::EC2::CapacityReservation":
		// arn:${Partition}:ec2:${Region}:${Account}:capacity-reservation/${CapacityReservationId}
		return standardFormat("ec2", "capacity-reservation", input, physicalId)
	case "AWS::EC2::CapacityReservationFleet":
		// arn:${Partition}:ec2:${Region}:${Account}:capacity-reservation-fleet/${CapacityReservationFleetId}
		return standardFormat("ec2", "capacity-reservation-fleet", input, physicalId)
	case "AWS::EC2::CarrierGateway":
		// arn:${Partition}:ec2:${Region}:${Account}:carrier-gateway/${CarrierGatewayId}
		return standardFormat("ec2", "carrier-gateway", input, physicalId)
	case "AWS::EC2::ClientVpnAuthorizationRule":
		return ""
	case "AWS::EC2::ClientVpnEndpoint":
		// arn:${Partition}:ec2:${Region}:${Account}:client-vpn-endpoint/${ClientVpnEndpointId}
		return standardFormat("ec2", "client-vpn-endpoint", input, physicalId)
	case "AWS::EC2::ClientVpnRoute":
		return ""
	case "AWS::EC2::ClientVpnTargetNetworkAssociation":
		return ""
	case "AWS::EC2::CustomerGateway":
		// arn:${Partition}:ec2:${Region}:${Account}:customer-gateway/${CustomerGatewayId}
		return standardFormat("ec2", "customer-gateway", input, physicalId)
	case "AWS::EC2::DHCPOptions":
		// arn:${Partition}:ec2:${Region}:${Account}:dhcp-options/${DhcpOptionsId}
		return standardFormat("ec2", "dhcp-options", input, physicalId)
	case "AWS::EC2::EC2Fleet":
		return ""
	case "AWS::EC2::EgressOnlyInternetGateway":
		// arn:${Partition}:ec2:${Region}:${Account}:egress-only-internet-gateway/${EgressOnlyInternetGatewayId}
		return standardFormat("ec2", "egress-only-internet-gateway", input, physicalId)
	case "AWS::EC2::EIP":
		// arn:${Partition}:ec2:${Region}:${Account}:elastic-ip/${AllocationId}
		return standardFormat("ec2", "elastic-ip", input, physicalId)
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
		// arn:${Partition}:ec2:${Region}:${Account}:instance/${InstanceId}
		return standardFormat("ec2", "instance", input, physicalId)
	case "AWS::EC2::InternetGateway":
		// arn:${Partition}:ec2:${Region}:${Account}:internet-gateway/${InternetGatewayId}
		return standardFormat("ec2", "internet-gateway", input, physicalId)
	case "AWS::EC2::IPAM":
		// arn:${Partition}:ec2::${Account}:ipam/${IpamId}
		return standardFormat("ec2", "ipam", input, physicalId)
	case "AWS::EC2::IPAMAllocation":
		return ""
	case "AWS::EC2::IPAMPool":
		// arn:${Partition}:ec2::${Account}:ipam-pool/${IpamPoolId}
		return standardFormat("ec2", "ipam-pool", input, physicalId)
	case "AWS::EC2::IPAMPoolCidr":
		return ""
	case "AWS::EC2::IPAMResourceDiscovery":
		// arn:${Partition}:ec2::${Account}:ipam-resource-discovery/${IpamResourceDiscoveryId}
		return standardFormat("ec2", "ipam-resource-discovery", input, physicalId)
	case "AWS::EC2::IPAMResourceDiscoveryAssociation":
		// arn:${Partition}:ec2::${Account}:ipam-resource-discovery-association/${IpamResourceDiscoveryAssociationId}
		return standardFormat("ec2", "ipam-resource-discovery-association", input, physicalId)
	case "AWS::EC2::IPAMScope":
		// arn:${Partition}:ec2::${Account}:ipam-scope/${IpamScopeId}
		return standardFormat("ec2", "ipam-scope", input, physicalId)
	case "AWS::EC2::KeyPair":
		// arn:${Partition}:ec2:${Region}:${Account}:key-pair/${KeyPairName}
		return standardFormat("ec2", "key-pair", input, physicalId)
	case "AWS::EC2::LaunchTemplate":
		// arn:${Partition}:ec2:${Region}:${Account}:launch-template/${LaunchTemplateId}
		return standardFormat("ec2", "launch-template", input, physicalId)
	case "AWS::EC2::LocalGatewayRoute":
		return ""
	case "AWS::EC2::LocalGatewayRouteTable":
		// arn:${Partition}:ec2:${Region}:${Account}:local-gateway-route-table/${LocalGatewayRouteTableId}
		return standardFormat("ec2", "local-gateway-route-table", input, physicalId)
	case "AWS::EC2::LocalGatewayRouteTableVirtualInterfaceGroupAssociation":
		// arn:${Partition}:ec2:${Region}:${Account}:local-gateway-route-table-virtual-interface-group-association/${LocalGatewayRouteTableVirtualInterfaceGroupAssociationId}
		return standardFormat("ec2", "local-gateway-route-table-virtual-interface-group-association", input, physicalId)
	case "AWS::EC2::LocalGatewayRouteTableVPCAssociation":
		// arn:${Partition}:ec2:${Region}:${Account}:local-gateway-route-table-vpc-association/${LocalGatewayRouteTableVpcAssociationId}
		return standardFormat("ec2", "local-gateway-route-table-vpc-association", input, physicalId)
	case "AWS::EC2::NatGateway":
		// arn:${Partition}:ec2:${Region}:${Account}:nat-gateway/${NatGatewayId}
		return standardFormat("ec2", "nat-gateway", input, physicalId)
	case "AWS::EC2::NetworkAcl":
		// arn:${Partition}:ec2:${Region}:${Account}:network-acl/${NaclId}
		return standardFormat("ec2", "network-acl", input, physicalId)
	case "AWS::EC2::NetworkAclEntry":
		return ""
	case "AWS::EC2::NetworkInsightsAccessScope":
		// arn:${Partition}:ec2:${Region}:${Account}:network-insights-access-scope/${NetworkInsightsAccessScopeId}
		return standardFormat("ec2", "network-insights-access-scope", input, physicalId)
	case "AWS::EC2::NetworkInsightsAccessScopeAnalysis":
		// arn:${Partition}:ec2:${Region}:${Account}:network-insights-access-scope-analysis/${NetworkInsightsAccessScopeAnalysisId}
		return standardFormat("ec2", "network-insights-access-scope-analysis", input, physicalId)
	case "AWS::EC2::NetworkInsightsAnalysis":
		// arn:${Partition}:ec2:${Region}:${Account}:network-insights-analysis/${NetworkInsightsAnalysisId}
		return standardFormat("ec2", "network-insights-analysis", input, physicalId)
	case "AWS::EC2::NetworkInsightsPath":
		// arn:${Partition}:ec2:${Region}:${Account}:network-insights-path/${NetworkInsightsPathId}
		return standardFormat("ec2", "network-insights-path", input, physicalId)
	case "AWS::EC2::NetworkInterface":
		// arn:${Partition}:ec2:${Region}:${Account}:network-interface/${NetworkInterfaceId}
		return standardFormat("ec2", "network-interface", input, physicalId)
	case "AWS::EC2::NetworkInterfaceAttachment":
		return ""
	case "AWS::EC2::NetworkInterfacePermission":
		return ""
	case "AWS::EC2::NetworkPerformanceMetricSubscription":
		return ""
	case "AWS::EC2::PlacementGroup":
		// arn:${Partition}:ec2:${Region}:${Account}:placement-group/${PlacementGroupName}
		return standardFormat("ec2", "placement-group", input, physicalId)
	case "AWS::EC2::PrefixList":
		// arn:${Partition}:ec2:${Region}:${Account}:prefix-list/${PrefixListId}
		return standardFormat("ec2", "prefix-list", input, physicalId)
	case "AWS::EC2::Route":
		return ""
	case "AWS::EC2::RouteTable":
		// arn:${Partition}:ec2:${Region}:${Account}:route-table/${RouteTableId}
		return standardFormat("ec2", "route-table", input, physicalId)
	case "AWS::EC2::SecurityGroup":
		// arn:${Partition}:ec2:${Region}:${Account}:security-group/${SecurityGroupId}
		return standardFormat("ec2", "security-group", input, physicalId)
	case "AWS::EC2::SecurityGroupEgress":
		return ""
	case "AWS::EC2::SecurityGroupIngress":
		return ""
	case "AWS::EC2::SpotFleet":
		// arn:${Partition}:ec2:${Region}:${Account}:spot-fleet-request/${SpotFleetRequestId}
		return standardFormat("ec2", "spot-fleet-request", input, physicalId)
	case "AWS::EC2::Subnet":
		// arn:${Partition}:ec2:${Region}:${Account}:subnet/${SubnetId}
		return standardFormat("ec2", "subnet", input, physicalId)
	case "AWS::EC2::SubnetCidrBlock":
		return ""
	case "AWS::EC2::SubnetNetworkAclAssociation":
		return ""
	case "AWS::EC2::SubnetRouteTableAssociation":
		return ""
	case "AWS::EC2::TrafficMirrorFilter":
		// arn:${Partition}:ec2:${Region}:${Account}:traffic-mirror-filter/${TrafficMirrorFilterId}
		return standardFormat("ec2", "traffic-mirror-filter", input, physicalId)
	case "AWS::EC2::TrafficMirrorFilterRule":
		// arn:${Partition}:ec2:${Region}:${Account}:traffic-mirror-filter-rule/${TrafficMirrorFilterRuleId}
		return standardFormat("ec2", "traffic-mirror-filter-rule", input, physicalId)
	case "AWS::EC2::TrafficMirrorSession":
		// arn:${Partition}:ec2:${Region}:${Account}:traffic-mirror-session/${TrafficMirrorSessionId}
		return standardFormat("ec2", "traffic-mirror-session", input, physicalId)
	case "AWS::EC2::TrafficMirrorTarget":
		// arn:${Partition}:ec2:${Region}:${Account}:traffic-mirror-target/${TrafficMirrorTargetId}
		return standardFormat("ec2", "traffic-mirror-target", input, physicalId)
	case "AWS::EC2::TransitGateway":
		// arn:${Partition}:ec2:${Region}:${Account}:transit-gateway/${TransitGatewayId}
		return standardFormat("ec2", "transit-gateway", input, physicalId)
	case "AWS::EC2::TransitGatewayAttachment":
		// arn:${Partition}:ec2:${Region}:${Account}:transit-gateway-attachment/${TransitGatewayAttachmentId}
		return standardFormat("ec2", "transit-gateway-attachment", input, physicalId)
	case "AWS::EC2::TransitGatewayConnect":
		return ""
	case "AWS::EC2::TransitGatewayMulticastDomain":
		// arn:${Partition}:ec2:${Region}:${Account}:transit-gateway-multicast-domain/${TransitGatewayMulticastDomainId}
		return standardFormat("ec2", "transit-gateway-multicast-domain", input, physicalId)
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
		// arn:${Partition}:ec2:${Region}:${Account}:transit-gateway-route-table/${TransitGatewayRouteTableId}
		return standardFormat("ec2", "transit-gateway-route-table", input, physicalId)
	case "AWS::EC2::TransitGatewayRouteTableAssociation":
		return ""
	case "AWS::EC2::TransitGatewayRouteTablePropagation":
		return ""
	case "AWS::EC2::TransitGatewayVpcAttachment":
		return ""
	case "AWS::EC2::VerifiedAccessEndpoint":
		// arn:${Partition}:ec2:${Region}:${Account}:verified-access-endpoint/${VerifiedAccessEndpointId}
		return standardFormat("ec2", "verified-access-endpoint", input, physicalId)
	case "AWS::EC2::VerifiedAccessGroup":
		// arn:${Partition}:ec2:${Region}:${Account}:verified-access-group/${VerifiedAccessGroupId}
		return standardFormat("ec2", "verified-access-group", input, physicalId)
	case "AWS::EC2::VerifiedAccessInstance":
		// arn:${Partition}:ec2:${Region}:${Account}:verified-access-instance/${VerifiedAccessInstanceId}
		return standardFormat("ec2", "verified-access-instance", input, physicalId)
	case "AWS::EC2::VerifiedAccessTrustProvider":
		// arn:${Partition}:ec2:${Region}:${Account}:verified-access-trust-provider/${VerifiedAccessTrustProviderId}
		return standardFormat("ec2", "verified-access-trust-provider", input, physicalId)
	case "AWS::EC2::Volume":
		// arn:${Partition}:ec2:${Region}:${Account}:volume/${VolumeId}
		return standardFormat("ec2", "volume", input, physicalId)
	case "AWS::EC2::VolumeAttachment":
		return ""
	case "AWS::EC2::VPC":
		// arn:${Partition}:ec2:${Region}:${Account}:vpc/${VpcId}
		return standardFormat("ec2", "vpc", input, physicalId)
	case "AWS::EC2::VPCCidrBlock":
		return ""
	case "AWS::EC2::VPCDHCPOptionsAssociation":
		return ""
	case "AWS::EC2::VPCEndpoint":
		// arn:${Partition}:ec2:${Region}:${Account}:vpc-endpoint/${VpcEndpointId}
		return standardFormat("ec2", "vpc-endpoint", input, physicalId)
	case "AWS::EC2::VPCEndpointConnectionNotification":
		return ""
	case "AWS::EC2::VPCEndpointService":
		// arn:${Partition}:ec2:${Region}:${Account}:vpc-endpoint-service/${VpcEndpointServiceId}
		return standardFormat("ec2", "vpc-endpoint-service", input, physicalId)
	case "AWS::EC2::VPCEndpointServicePermissions":
		// arn:${Partition}:ec2:${Region}:${Account}:vpc-endpoint-service-permission/${VpcEndpointServicePermissionId}
		return standardFormat("ec2", "vpc-endpoint-service-permission", input, physicalId)
	case "AWS::EC2::VPCGatewayAttachment":
		return ""
	case "AWS::EC2::VPCPeeringConnection":
		// arn:${Partition}:ec2:${Region}:${Account}:vpc-peering-connection/${VpcPeeringConnectionId}
		return standardFormat("ec2", "vpc-peering-connection", input, physicalId)
	case "AWS::EC2::VPNConnection":
		// arn:${Partition}:ec2:${Region}:${Account}:vpn-connection/${VpnConnectionId}
		return standardFormat("ec2", "vpn-connection", input, physicalId)
	case "AWS::EC2::VPNConnectionRoute":
		return ""
	case "AWS::EC2::VPNGateway":
		// arn:${Partition}:ec2:${Region}:${Account}:vpn-gateway/${VpnGatewayId}
		return standardFormat("ec2", "vpn-gateway", input, physicalId)
	case "AWS::EC2::VPNGatewayRoutePropagation":
		return ""
	case "AWS::Lambda::Alias":
		// arn:${Partition}:lambda:${Region}:${Account}:function:${FunctionName}:${Alias}
		// Get the function name and alias from the template
		_, props, _ := s11n.GetMapValue(input.resource, "Properties")
		if props == nil {
			config.Debugf("unexpected, AWS::Lambda::Alias props are nil: %v",
				node.ToJson(input.resource))
			return ""
		}
		_, functionName, _ := s11n.GetMapValue(props, "FunctionName")
		if functionName == nil {
			config.Debugf("unexpected, AWS::Lambda::Alias FunctionName is nil: %v",
				node.ToJson(input.resource))
			return ""
		}
		_, name, _ := s11n.GetMapValue(props, "Name")
		if name == nil {
			config.Debugf("unexpected, AWS::Lambda::Alias Name is nil: %v",
				node.ToJson(input.resource))
			return ""
		}
		fn := functionName.Value
		if fn == "" {
			fn = physicalId
			if fn == "" {
				fn = "function-name"
			}
		}
		return fmt.Sprintf("arn:%v:lambda:%v:%v:function:%v:%v",
			input.env.partition, input.env.region, input.env.account,
			fn, name.Value)
	case "AWS::Lambda::CodeSigningConfig":
		// arn:${Partition}:lambda:${Region}:${Account}:code-signing-config:${CodeSigningConfigId}
		return colonFormat("lambda", "code-signing-config", input, physicalId)
	case "AWS::Lambda::EventInvokeConfig":
		return ""
	case "AWS::Lambda::EventSourceMapping":
		// arn:${Partition}:lambda:${Region}:${Account}:event-source-mapping:${UUID}
		return colonFormat("lambda", "event-source-mapping", input, physicalId)
	case "AWS::Lambda::Function":
		// arn:${Partition}:lambda:${Region}:${Account}:function:${FunctionName}
		return colonFormat("lambda", "function", input, physicalId)
	case "AWS::Lambda::LayerVersion":
		// arn:${Partition}:lambda:${Region}:${Account}:layer:${LayerName}:${LayerVersion}
		_, props, _ := s11n.GetMapValue(input.resource, "Properties")
		if props == nil {
			config.Debugf("unexpected, AWS::Lambda::Alias props are nil: %v",
				node.ToJson(input.resource))
			return ""
		}
		_, layerNameProp, _ := s11n.GetMapValue(props, "LayerName")
		layerName := ""
		if layerNameProp != nil {
			layerName = layerNameProp.Value
		}
		if layerName == "" {
			layerName = physicalId
			if layerName == "" {
				layerName = "layername"
			}
		}
		return fmt.Sprintf("arn:%v:lambda:%v:%v:layer:%v:%v",
			input.env.partition, input.env.region, input.env.account,
			layerName, "1")
	case "AWS::Lambda::LayerVersionPermission":
		return ""
	case "AWS::Lambda::Permission":
		return ""
	case "AWS::Lambda::Url":
		return ""
	case "AWS::Lambda::Version":
		// arn:${Partition}:lambda:${Region}:${Account}:function:${FunctionName}:${Version}
		_, props, _ := s11n.GetMapValue(input.resource, "Properties")
		if props == nil {
			config.Debugf("unexpected, AWS::Lambda::Version props are nil: %v",
				node.ToJson(input.resource))
			return ""
		}
		_, functionName, _ := s11n.GetMapValue(props, "FunctionName")
		if functionName == nil {
			config.Debugf("unexpected, AWS::Lambda::Version FunctionName is nil: %v",
				node.ToJson(input.resource))
			return ""
		}
		fn := functionName.Value
		if fn == "" {
			fn = physicalId
			if fn == "" {
				fn = "function-name"
			}
		}
		return fmt.Sprintf("arn:%v:lambda:%v:%v:function:%v:%v",
			input.env.partition, input.env.region, input.env.account,
			fn, "1")
	case "AWS::IAM::AccessKey":
		return ""
	case "AWS::IAM::Group":
		// arn:${Partition}:iam::${Account}:group/${GroupNameWithPath}
		return globalFormat("iam", "group", input, physicalId)
	case "AWS::IAM::GroupPolicy":
		// arn:${Partition}:iam::${Account}:policy/${PolicyNameWithPath}
		return globalFormat("iam", "policy", input, physicalId)
	case "AWS::IAM::InstanceProfile":
		// arn:${Partition}:iam::${Account}:instance-profile/${InstanceProfileNameWithPath}
		return globalFormat("iam", "instance-profile", input, physicalId)
	case "AWS::IAM::ManagedPolicy":
		// arn:${Partition}:iam::${Account}:policy/${PolicyNameWithPath}
		return globalFormat("iam", "policy", input, physicalId)
	case "AWS::IAM::OIDCProvider":
		// arn:${Partition}:iam::${Account}:oidc-provider/${OidcProviderName}
		return globalFormat("iam", "oidc-provider", input, physicalId)
	case "AWS::IAM::Policy":
		// arn:${Partition}:iam::${Account}:policy/${PolicyNameWithPath}
		return globalFormat("iam", "policy", input, physicalId)
	case "AWS::IAM::Role":
		// arn:${Partition}:iam::${Account}:role/${RoleNameWithPath}
		return globalFormat("iam", "role", input, physicalId)
	case "AWS::IAM::RolePolicy":
		return ""
	case "AWS::IAM::SAMLProvider":
		// arn:${Partition}:iam::${Account}:saml-provider/${SamlProviderName}
		return globalFormat("iam", "saml-provider", input, physicalId)
	case "AWS::IAM::ServerCertificate":
		// arn:${Partition}:iam::${Account}:server-certificate/${CertificateNameWithPath}
		return globalFormat("iam", "server-provider", input, physicalId)
	case "AWS::IAM::ServiceLinkedRole":
		return ""
	case "AWS::IAM::User":
		// arn:${Partition}:iam::${Account}:user/${UserNameWithPath}
		return globalFormat("iam", "user", input, physicalId)
	case "AWS::IAM::UserPolicy":
		// arn:${Partition}:iam::${Account}:policy/${PolicyNameWithPath}
		return globalFormat("iam", "policy", input, physicalId)
	case "AWS::IAM::UserToGroupAddition":
		return ""
	case "AWS::IAM::VirtualMFADevice":
		return ""
	default:
		return ""
	}
}

// standardFormat returns an arn that fits the standard arn format.
// for example
// arn:${Partition}:ec2:${Region}:${Account}:capacity-reservation/${CapacityReservationId}
func standardFormat(service string, subType string, input PredictionInput, physicalId string) string {
	return fmt.Sprintf("arn:%v:%v:%v:%v:%v/%v",
		input.env.partition, service, input.env.region, input.env.account, subType, physicalId)
}

// colonFormat returns an arn that fits the (other) standard arn format.
// for example
// arn:${Partition}:ec2:${Region}:${Account}:capacity-reservation:${CapacityReservationId}
func colonFormat(service string, subType string, input PredictionInput, physicalId string) string {
	return fmt.Sprintf("arn:%v:%v:%v:%v:%v:%v",
		input.env.partition, service, input.env.region, input.env.account, subType, physicalId)
}

// GlobalFormat returns an arn that fits IAM and other non-regional services
// for example
// arn:${Partition}:iam::${Account}:group/${GroupNameWithPath}
func globalFormat(service string, subType string, input PredictionInput, physicalId string) string {
	return fmt.Sprintf("arn:%v:%v::%v:%v:%v",
		input.env.partition, service, input.env.account, subType, physicalId)
}
