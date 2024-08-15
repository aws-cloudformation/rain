package forecast

import (
	"github.com/aws-cloudformation/rain/internal/aws/acm"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	fc "github.com/aws-cloudformation/rain/plugins/forecast"
	"gopkg.in/yaml.v3"
)

// Elastic Load Balancing

// AWS::ElasticLoadBalancingV2::Listener

func CheckELBListener(input fc.PredictionInput) fc.Forecast {

	forecast := makeForecast(input.TypeName, input.LogicalId)

	// Resource handler returned message:
	// "Certificate 'arn:aws:acm:us-east-1:X:certificate/Y'
	//not found (Service: ElasticLoadBalancingV2
	// (Expired certs cause this error)

	// Get the certificate arn from props
	_, props, _ := s11n.GetMapValue(input.Resource, "Properties")
	if props == nil {
		config.Debugf("expected %s to have Properties", input.LogicalId)
		return forecast
	}

	_, certArns, _ := s11n.GetMapValue(props, "Certificates")
	if certArns == nil {
		config.Debugf("expected %s to have Certificates", input.LogicalId)
		return forecast
	}
	config.Debugf("Certificates: %s", node.ToSJson(certArns))

	spin(input.TypeName, input.LogicalId, "Checking ELB Certs")

	for _, certArnNode := range certArns.Content {
		if len(certArnNode.Content) != 2 {
			config.Debugf("Expected 2 children for certArnNode %s", node.ToSJson(certArnNode))
			continue
		}
		certArn := certArnNode.Content[1].Value
		config.Debugf("Checking certArn %s", certArn)
		ok, err := acm.CheckCertificate(certArn)
		if err != nil {
			config.Debugf("Error checking certArn %s: %s", certArn, err)
			spinner.Pop()
			return forecast
		}
		code := F0012
		if !ok {
			forecast.Add(code, false, "Certificate not found or expired")
		} else {
			forecast.Add(code, true, "Certificate found")
		}
	}

	spinner.Pop()

	return forecast
}

func CheckELBTargetGroup(input fc.PredictionInput) fc.Forecast {

	forecast := makeForecast(input.TypeName, input.LogicalId)

	// Check to make sure the Port and Protocol properties match
	portNode := input.GetPropertyNode("Port")
	protocolNode := input.GetPropertyNode("Protocol")
	if portNode != nil && protocolNode != nil {
		port := portNode.Value
		protocol := protocolNode.Value
		if port == "443" {
			if protocol == "HTTPS" {
				forecast.Add(F0014, true, "ELB target group port and protocol match")
			} else {
				forecast.Add(F0014, false, "ELB target group port and protocol do not match")
			}
		}
		if port == "80" {
			if protocol == "HTTP" {
				forecast.Add(F0014, true, "ELB target group port and protocol match")
			} else {
				forecast.Add(F0014, false, "ELB target group port and protocol do not match")
			}
		}
	}

	targetTypeNode := input.GetPropertyNode("TargetType")
	if targetTypeNode != nil {
		targetType := targetTypeNode.Value
		if targetType != "instance" {
			// If this target group is being used by an ASG, the type must be instance
			// Look at template resources to see if a launch template refers to this
			autoscalingGroups := input.Source.GetResourcesOfType("AWS::AutoScaling::AutoScalingGroup")
			for _, asg := range autoscalingGroups {
				_, props, _ := s11n.GetMapValue(asg, "Properties")
				if props == nil {
					continue
				}
				config.Debugf("Properties: %s", node.ToSJson(props))
				_, targetGroupARNs, _ := s11n.GetMapValue(props, "TargetGroupARNs")
				if targetGroupARNs == nil {
					continue
				}
				for _, targetGroupArn := range targetGroupARNs.Content {
					if targetGroupArn.Kind == yaml.MappingNode {
						if targetGroupArn.Content[0].Kind == yaml.ScalarNode &&
							targetGroupArn.Content[0].Value == "Ref" {
							if targetGroupArn.Content[1].Value == input.LogicalId {
								forecast.Add(F0015, false,
									"ELB target group must be of type instance if it is used by an ASG")
							} else {
								forecast.Add(F0015, true,
									"ELB target group is of type instance if it is used by an ASG")
							}
						}
					}
				}
			}
		}
	}

	return forecast
}
