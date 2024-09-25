package cloudfront

import (
	"context"
	"time"

	rainaws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

func getClient() *cloudfront.Client {
	return cloudfront.NewFromConfig(rainaws.Config())
}

func Invalidate(distributionId string) error {
	client := getClient()
	reference := aws.String(time.Now().String())
	input := &cloudfront.CreateInvalidationInput{
		DistributionId: aws.String(distributionId),
		InvalidationBatch: &types.InvalidationBatch{
			CallerReference: reference,
			Paths: &types.Paths{
				Quantity: aws.Int32(1),
				Items:    []string{"/*"},
			},
		},
	}

	// Keep calling the API with the same reference until it reports complete
	for i := 0; i < 10; i++ {
		resp, err := client.CreateInvalidation(context.Background(), input)
		if err != nil {
			return err
		}
		config.Debugf("CreateInvalidation response: %+v", resp)
		config.Debugf("CreateInvalidation Status: %s", *resp.Invalidation.Status)
		if *resp.Invalidation.Status == "Completed" {
			break
		}
		// 10 max iterations at 6 seconds each
		time.Sleep(6 * time.Second)
	}

	return nil
}
