package acm

import (
	"context"
	rainaws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"time"
)

func getClient() *acm.Client {
	return acm.NewFromConfig(rainaws.Config())
}

// CheckCertificate checks if the certificate exists and is valid
func CheckCertificate(arn string) (bool, error) {
	client := getClient()
	_, err := client.GetCertificate(context.Background(), &acm.GetCertificateInput{
		CertificateArn: &arn,
	})

	if err != nil {
		return false, err
	}

	res, err := client.DescribeCertificate(context.Background(), &acm.DescribeCertificateInput{
		CertificateArn: &arn,
	})
	if err != nil {
		return false, err
	}
	// Make sure the cert has not expired
	if res.Certificate.NotAfter.Before(time.Now()) {
		config.Debugf("Cert expired: %s", arn)
		return false, nil
	}

	return true, nil
}
