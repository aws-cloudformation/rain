package s3

import (
	"context"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var s3Client *s3.Client

func getClient() *s3.Client {
	if s3Client == nil {
		s3Client = s3.New(client.Config())
	}

	return s3Client
}

func BucketExists(bucketName string) bool {
	req := getClient().HeadBucketRequest(&s3.HeadBucketInput{
		Bucket: &bucketName,
	})

	_, err := req.Send(context.Background())

	return err == nil
}

func CreateBucket(bucketName string) client.Error {
	req := getClient().CreateBucketRequest(&s3.CreateBucketInput{
		Bucket: &bucketName,
	})

	_, err := req.Send(context.Background())

	return client.NewError(err)
}
