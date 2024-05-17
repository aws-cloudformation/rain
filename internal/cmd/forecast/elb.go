package forecast

import (
	"github.com/aws-cloudformation/rain/internal/aws/acm"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
)

// Elastic Load Balancing

// AWS::ElasticLoadBalancingV2::Listener

func checkELBListener(input PredictionInput) Forecast {

	forecast := makeForecast(input.typeName, input.logicalId)

	// Resource handler returned message:
	// "Certificate 'arn:aws:acm:us-east-1:X:certificate/Y'
	//not found (Service: ElasticLoadBalancingV2
	// (Expired certs cause this error)

	// Get the certificate arn from props
	_, props, _ := s11n.GetMapValue(input.resource, "Properties")
	if props == nil {
		config.Debugf("expected %s to have Properties", input.logicalId)
		return forecast
	}

	_, certArns, _ := s11n.GetMapValue(props, "Certificates")
	if certArns == nil {
		config.Debugf("expected %s to have Certificates", input.logicalId)
		return forecast
	}
	config.Debugf("Certificates: %s", node.ToSJson(certArns))

	spin(input.typeName, input.logicalId, "Checking ELB Certs")

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
