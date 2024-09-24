package cloudfront

import (
	"context"

	rainaws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
)

func getClient() *cloudfront.Client {
	return cloudfront.NewFromConfig(rainaws.Config())
}

func Invalidate(distributionId string) error {
	client := getClient()
	input := &cloudfront.CreateInvalidationInput{
		DistributionId: aws.String(distributionId),
	}
	resp, err := client.CreateInvalidation(context.Background(), input)
	if err != nil {
		return err
	}
	config.Debugf("CreateInvalidation response: %+v", resp)

	return nil
}
