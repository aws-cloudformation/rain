package kms

import (
	"context"
	rainaws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"time"
)

func getClient() *kms.Client {
	return kms.NewFromConfig(rainaws.Config())
}

func IsKeyArnValid(keyArn string) bool {
	client := getClient()

	// Use the aws go sdk to call kms and make sure the key is valid
	key, err := client.DescribeKey(context.Background(), &kms.DescribeKeyInput{KeyId: &keyArn})
	if err != nil {
		// Throws an error if it can't find the key
		return false
	}
	if !key.KeyMetadata.Enabled {
		return false
	}
	if key.KeyMetadata.ValidTo != nil {
		// Make sure the key is not expired
		if key.KeyMetadata.ValidTo.Before(time.Now()) {
			return false
		}
	}
	return true
}
