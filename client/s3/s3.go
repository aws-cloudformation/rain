package s3

import (
	"context"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func getClient() *s3.Client {
	return s3.NewFromConfig(client.Config())
}

// BucketExists checks whether the named bucket exists
func BucketExists(bucketName string) bool {
	_, err := getClient().HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: &bucketName,
	})

	return err == nil
}

// CreateBucket creates a new S3 bucket
func CreateBucket(bucketName string) error {
	_, err := getClient().CreateBucket(context.Background(), &s3.CreateBucketInput{
		Bucket: &bucketName,
	})

	return err
}
