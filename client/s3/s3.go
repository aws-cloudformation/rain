package s3

import (
	"context"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func getClient() *s3.Client {
	return s3.New(client.Config())
}

// BucketExists checks whether the named bucket exists
func BucketExists(bucketName string) bool {
	req := getClient().HeadBucketRequest(&s3.HeadBucketInput{
		Bucket: &bucketName,
	})

	_, err := req.Send(context.Background())

	return err == nil
}

// CreateBucket creates a new S3 bucket
func CreateBucket(bucketName string) client.Error {
	req := getClient().CreateBucketRequest(&s3.CreateBucketInput{
		Bucket: &bucketName,
	})

	_, err := req.Send(context.Background())

	return client.NewError(err)
}
