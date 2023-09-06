package logs

import (
	"fmt"
	"strings"

	_ "embed"
)

//go:embed chart-template.html
var template string

// createChart outputs an html file to stdout with a gantt chart
// that shows the durations for each resource of the latest stack action
func createChart(stackName string) error {

	data := ` [
                {
                    "Id": "ecs-cw-eval",
                    "Type": "AWS::CloudFormation::Stack",
                    "Timestamp": "2023-08-22T17:28:30.019000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "Service",
                    "Type": "AWS::ECS::Service",
                    "Timestamp": "2023-08-22T17:28:28.849000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "Service",
                    "Type": "AWS::ECS::Service",
                    "Timestamp": "2023-08-22T17:03:55.105000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "Service",
                    "Type": "AWS::ECS::Service",
                    "Timestamp": "2023-08-22T17:03:53.247000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "LoadBalancerListener",
                    "Type": "AWS::ElasticLoadBalancingV2::Listener",
                    "Timestamp": "2023-08-22T17:03:52.572000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "LoadBalancerListener",
                    "Type": "AWS::ElasticLoadBalancingV2::Listener",
                    "Timestamp": "2023-08-22T17:03:52.277000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "LoadBalancerListener",
                    "Type": "AWS::ElasticLoadBalancingV2::Listener",
                    "Timestamp": "2023-08-22T17:03:50.787000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "LoadBalancer",
                    "Type": "AWS::ElasticLoadBalancingV2::LoadBalancer",
                    "Timestamp": "2023-08-22T17:03:50.170000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PrivateSubnet1DefaultRoute",
                    "Type": "AWS::EC2::Route",
                    "Timestamp": "2023-08-22T17:03:43.595000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PrivateSubnet1DefaultRoute",
                    "Type": "AWS::EC2::Route",
                    "Timestamp": "2023-08-22T17:03:43.269000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet1DefaultRoute",
                    "Type": "AWS::EC2::Route",
                    "Timestamp": "2023-08-22T17:03:41.125000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1NATGateway",
                    "Type": "AWS::EC2::NatGateway",
                    "Timestamp": "2023-08-22T17:03:40.461000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PrivateSubnet2DefaultRoute",
                    "Type": "AWS::EC2::Route",
                    "Timestamp": "2023-08-22T17:03:14.939000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PrivateSubnet2DefaultRoute",
                    "Type": "AWS::EC2::Route",
                    "Timestamp": "2023-08-22T17:03:14.708000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet2DefaultRoute",
                    "Type": "AWS::EC2::Route",
                    "Timestamp": "2023-08-22T17:03:12.817000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet2NATGateway",
                    "Type": "AWS::EC2::NatGateway",
                    "Timestamp": "2023-08-22T17:03:12.115000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PublicSubnet1NATGateway",
                    "Type": "AWS::EC2::NatGateway",
                    "Timestamp": "2023-08-22T17:01:49.687000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "LoadBalancer",
                    "Type": "AWS::ElasticLoadBalancingV2::LoadBalancer",
                    "Timestamp": "2023-08-22T17:01:48.886000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1NATGateway",
                    "Type": "AWS::EC2::NatGateway",
                    "Timestamp": "2023-08-22T17:01:48.492000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "LoadBalancer",
                    "Type": "AWS::ElasticLoadBalancingV2::LoadBalancer",
                    "Timestamp": "2023-08-22T17:01:48.087000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet2NATGateway",
                    "Type": "AWS::EC2::NatGateway",
                    "Timestamp": "2023-08-22T17:01:47.534000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1RouteTableAssociation",
                    "Type": "AWS::EC2::SubnetRouteTableAssociation",
                    "Timestamp": "2023-08-22T17:01:47.431000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PublicSubnet2NATGateway",
                    "Type": "AWS::EC2::NatGateway",
                    "Timestamp": "2023-08-22T17:01:46.050000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet2DefaultRoute",
                    "Type": "AWS::EC2::Route",
                    "Timestamp": "2023-08-22T17:01:44.931000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PublicSubnet1DefaultRoute",
                    "Type": "AWS::EC2::Route",
                    "Timestamp": "2023-08-22T17:01:44.638000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PublicSubnet2DefaultRoute",
                    "Type": "AWS::EC2::Route",
                    "Timestamp": "2023-08-22T17:01:44.577000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1DefaultRoute",
                    "Type": "AWS::EC2::Route",
                    "Timestamp": "2023-08-22T17:01:44.357000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "TaskExecutionPolicy",
                    "Type": "AWS::IAM::Policy",
                    "Timestamp": "2023-08-22T17:01:43.366000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PublicSubnet2DefaultRoute",
                    "Type": "AWS::EC2::Route",
                    "Timestamp": "2023-08-22T17:01:43.080000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1DefaultRoute",
                    "Type": "AWS::EC2::Route",
                    "Timestamp": "2023-08-22T17:01:43.049000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet1RouteTableAssociation",
                    "Type": "AWS::EC2::SubnetRouteTableAssociation",
                    "Timestamp": "2023-08-22T17:01:42.579000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "VPCGW",
                    "Type": "AWS::EC2::VPCGatewayAttachment",
                    "Timestamp": "2023-08-22T17:01:42.485000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PrivateSubnet1RouteTableAssociation",
                    "Type": "AWS::EC2::SubnetRouteTableAssociation",
                    "Timestamp": "2023-08-22T17:01:42.272000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet1RouteTableAssociation",
                    "Type": "AWS::EC2::SubnetRouteTableAssociation",
                    "Timestamp": "2023-08-22T17:01:40.768000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "TargetGroup",
                    "Type": "AWS::ElasticLoadBalancingV2::TargetGroup",
                    "Timestamp": "2023-08-22T17:01:40.498000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PrivateSubnet1RouteTable",
                    "Type": "AWS::EC2::RouteTable",
                    "Timestamp": "2023-08-22T17:01:40.210000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PrivateSubnet2RouteTableAssociation",
                    "Type": "AWS::EC2::SubnetRouteTableAssociation",
                    "Timestamp": "2023-08-22T17:01:37.290000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PublicSubnet2RouteTableAssociation",
                    "Type": "AWS::EC2::SubnetRouteTableAssociation",
                    "Timestamp": "2023-08-22T17:01:37.145000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PrivateSubnet2RouteTableAssociation",
                    "Type": "AWS::EC2::SubnetRouteTableAssociation",
                    "Timestamp": "2023-08-22T17:01:37.047000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet2RouteTableAssociation",
                    "Type": "AWS::EC2::SubnetRouteTableAssociation",
                    "Timestamp": "2023-08-22T17:01:36.898000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1RouteTableAssociation",
                    "Type": "AWS::EC2::SubnetRouteTableAssociation",
                    "Timestamp": "2023-08-22T17:01:36.865000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet2RouteTableAssociation",
                    "Type": "AWS::EC2::SubnetRouteTableAssociation",
                    "Timestamp": "2023-08-22T17:01:35.645000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet2RouteTableAssociation",
                    "Type": "AWS::EC2::SubnetRouteTableAssociation",
                    "Timestamp": "2023-08-22T17:01:35.499000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1RouteTableAssociation",
                    "Type": "AWS::EC2::SubnetRouteTableAssociation",
                    "Timestamp": "2023-08-22T17:01:35.433000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet2RouteTable",
                    "Type": "AWS::EC2::RouteTable",
                    "Timestamp": "2023-08-22T17:01:35.125000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PublicSubnet2RouteTable",
                    "Type": "AWS::EC2::RouteTable",
                    "Timestamp": "2023-08-22T17:01:34.942000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PublicSubnet1RouteTable",
                    "Type": "AWS::EC2::RouteTable",
                    "Timestamp": "2023-08-22T17:01:34.850000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "LoadBalancerEgress",
                    "Type": "AWS::EC2::SecurityGroupEgress",
                    "Timestamp": "2023-08-22T17:01:34.044000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "Ingress",
                    "Type": "AWS::EC2::SecurityGroupIngress",
                    "Timestamp": "2023-08-22T17:01:33.554000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "LoadBalancerEgress",
                    "Type": "AWS::EC2::SecurityGroupEgress",
                    "Timestamp": "2023-08-22T17:01:33.336000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "Ingress",
                    "Type": "AWS::EC2::SecurityGroupIngress",
                    "Timestamp": "2023-08-22T17:01:32.954000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "Ingress",
                    "Type": "AWS::EC2::SecurityGroupIngress",
                    "Timestamp": "2023-08-22T17:01:32.758000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "LoadBalancerEgress",
                    "Type": "AWS::EC2::SecurityGroupEgress",
                    "Timestamp": "2023-08-22T17:01:32.706000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "ServiceSecurityGroup",
                    "Type": "AWS::EC2::SecurityGroup",
                    "Timestamp": "2023-08-22T17:01:32.233000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "ServiceSecurityGroup",
                    "Type": "AWS::EC2::SecurityGroup",
                    "Timestamp": "2023-08-22T17:01:31.231000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "LoadBalancerSecurityGroup",
                    "Type": "AWS::EC2::SecurityGroup",
                    "Timestamp": "2023-08-22T17:01:29.113000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "TaskDefinition",
                    "Type": "AWS::ECS::TaskDefinition",
                    "Timestamp": "2023-08-22T17:01:28.142000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "LoadBalancerSecurityGroup",
                    "Type": "AWS::EC2::SecurityGroup",
                    "Timestamp": "2023-08-22T17:01:28.081000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "TaskDefinition",
                    "Type": "AWS::ECS::TaskDefinition",
                    "Timestamp": "2023-08-22T17:01:27.821000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet2",
                    "Type": "AWS::EC2::Subnet",
                    "Timestamp": "2023-08-22T17:01:27.707000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "TaskExecutionPolicy",
                    "Type": "AWS::IAM::Policy",
                    "Timestamp": "2023-08-22T17:01:27.593000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1",
                    "Type": "AWS::EC2::Subnet",
                    "Timestamp": "2023-08-22T17:01:27.453000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "VPCGW",
                    "Type": "AWS::EC2::VPCGatewayAttachment",
                    "Timestamp": "2023-08-22T17:01:27.123000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet2Subnet",
                    "Type": "AWS::EC2::Subnet",
                    "Timestamp": "2023-08-22T17:01:27.042000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PrivateSubnet1Subnet",
                    "Type": "AWS::EC2::Subnet",
                    "Timestamp": "2023-08-22T17:01:26.970000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PublicSubnet2EIP",
                    "Type": "AWS::EC2::EIP",
                    "Timestamp": "2023-08-22T17:01:26.761000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "VPCGW",
                    "Type": "AWS::EC2::VPCGatewayAttachment",
                    "Timestamp": "2023-08-22T17:01:26.727000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1EIP",
                    "Type": "AWS::EC2::EIP",
                    "Timestamp": "2023-08-22T17:01:26.592000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "TaskExecutionPolicy",
                    "Type": "AWS::IAM::Policy",
                    "Timestamp": "2023-08-22T17:01:26.486000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "TaskDefinition",
                    "Type": "AWS::ECS::TaskDefinition",
                    "Timestamp": "2023-08-22T17:01:26.449000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "ServiceSecurityGroup",
                    "Type": "AWS::EC2::SecurityGroup",
                    "Timestamp": "2023-08-22T17:01:26.387000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "InternetGateway",
                    "Type": "AWS::EC2::InternetGateway",
                    "Timestamp": "2023-08-22T17:01:26.253000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "TaskRole",
                    "Type": "AWS::IAM::Role",
                    "Timestamp": "2023-08-22T17:01:25.592000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "TaskExecutionRole",
                    "Type": "AWS::IAM::Role",
                    "Timestamp": "2023-08-22T17:01:25.464000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "TargetGroup",
                    "Type": "AWS::ElasticLoadBalancingV2::TargetGroup",
                    "Timestamp": "2023-08-22T17:01:24.840000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet2RouteTable",
                    "Type": "AWS::EC2::RouteTable",
                    "Timestamp": "2023-08-22T17:01:24.585000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1",
                    "Type": "AWS::EC2::Subnet",
                    "Timestamp": "2023-08-22T17:01:24.577000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet2",
                    "Type": "AWS::EC2::Subnet",
                    "Timestamp": "2023-08-22T17:01:24.566000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet2RouteTable",
                    "Type": "AWS::EC2::RouteTable",
                    "Timestamp": "2023-08-22T17:01:24.453000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet2Subnet",
                    "Type": "AWS::EC2::Subnet",
                    "Timestamp": "2023-08-22T17:01:24.442000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet1RouteTable",
                    "Type": "AWS::EC2::RouteTable",
                    "Timestamp": "2023-08-22T17:01:24.438000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1RouteTable",
                    "Type": "AWS::EC2::RouteTable",
                    "Timestamp": "2023-08-22T17:01:24.342000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet1Subnet",
                    "Type": "AWS::EC2::Subnet",
                    "Timestamp": "2023-08-22T17:01:24.331000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "LoadBalancerSecurityGroup",
                    "Type": "AWS::EC2::SecurityGroup",
                    "Timestamp": "2023-08-22T17:01:23.316000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1",
                    "Type": "AWS::EC2::Subnet",
                    "Timestamp": "2023-08-22T17:01:23.313000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet2RouteTable",
                    "Type": "AWS::EC2::RouteTable",
                    "Timestamp": "2023-08-22T17:01:23.294000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet2RouteTable",
                    "Type": "AWS::EC2::RouteTable",
                    "Timestamp": "2023-08-22T17:01:23.275000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet2",
                    "Type": "AWS::EC2::Subnet",
                    "Timestamp": "2023-08-22T17:01:23.271000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet2Subnet",
                    "Type": "AWS::EC2::Subnet",
                    "Timestamp": "2023-08-22T17:01:23.266000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet1RouteTable",
                    "Type": "AWS::EC2::RouteTable",
                    "Timestamp": "2023-08-22T17:01:23.260000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1RouteTable",
                    "Type": "AWS::EC2::RouteTable",
                    "Timestamp": "2023-08-22T17:01:23.253000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "TargetGroup",
                    "Type": "AWS::ElasticLoadBalancingV2::TargetGroup",
                    "Timestamp": "2023-08-22T17:01:23.250000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PrivateSubnet1Subnet",
                    "Type": "AWS::EC2::Subnet",
                    "Timestamp": "2023-08-22T17:01:23.245000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "VPC",
                    "Type": "AWS::EC2::VPC",
                    "Timestamp": "2023-08-22T17:01:22.707000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "Cluster",
                    "Type": "AWS::ECS::Cluster",
                    "Timestamp": "2023-08-22T17:01:14.292000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "VPC",
                    "Type": "AWS::EC2::VPC",
                    "Timestamp": "2023-08-22T17:01:11.168000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "Logs",
                    "Type": "AWS::Logs::LogGroup",
                    "Timestamp": "2023-08-22T17:01:11.028000+00:00",
                    "Status": "CREATE_COMPLETE"
                },
                {
                    "Id": "PublicSubnet2EIP",
                    "Type": "AWS::EC2::EIP",
                    "Timestamp": "2023-08-22T17:01:11.011000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1EIP",
                    "Type": "AWS::EC2::EIP",
                    "Timestamp": "2023-08-22T17:01:10.831000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "Cluster",
                    "Type": "AWS::ECS::Cluster",
                    "Timestamp": "2023-08-22T17:01:10.792000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "Logs",
                    "Type": "AWS::Logs::LogGroup",
                    "Timestamp": "2023-08-22T17:01:10.742000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "InternetGateway",
                    "Type": "AWS::EC2::InternetGateway",
                    "Timestamp": "2023-08-22T17:01:10.718000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "TaskExecutionRole",
                    "Type": "AWS::IAM::Role",
                    "Timestamp": "2023-08-22T17:01:09.951000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "TaskRole",
                    "Type": "AWS::IAM::Role",
                    "Timestamp": "2023-08-22T17:01:09.938000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet2EIP",
                    "Type": "AWS::EC2::EIP",
                    "Timestamp": "2023-08-22T17:01:09.719000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "VPC",
                    "Type": "AWS::EC2::VPC",
                    "Timestamp": "2023-08-22T17:01:09.688000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "TaskExecutionRole",
                    "Type": "AWS::IAM::Role",
                    "Timestamp": "2023-08-22T17:01:09.650000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "Cluster",
                    "Type": "AWS::ECS::Cluster",
                    "Timestamp": "2023-08-22T17:01:09.641000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "PublicSubnet1EIP",
                    "Type": "AWS::EC2::EIP",
                    "Timestamp": "2023-08-22T17:01:09.634000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "Logs",
                    "Type": "AWS::Logs::LogGroup",
                    "Timestamp": "2023-08-22T17:01:09.632000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "InternetGateway",
                    "Type": "AWS::EC2::InternetGateway",
                    "Timestamp": "2023-08-22T17:01:09.629000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "TaskRole",
                    "Type": "AWS::IAM::Role",
                    "Timestamp": "2023-08-22T17:01:09.609000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "ecs-cw-eval",
                    "Type": "AWS::CloudFormation::Stack",
                    "Timestamp": "2023-08-22T17:01:07.030000+00:00",
                    "Status": "CREATE_IN_PROGRESS"
                },
                {
                    "Id": "ecs-cw-eval",
                    "Type": "AWS::CloudFormation::Stack",
                    "Timestamp": "2023-08-22T17:00:59.245000+00:00",
                    "Status": "REVIEW_IN_PROGRESS"
                }
            ];
	`

	rendered := strings.Replace(template, "__DATA__", data, 1)

	fmt.Println(rendered)

	return nil
}
