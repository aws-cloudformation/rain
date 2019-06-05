package s3

import (
	"github.com/aws-cloudformation/rain/client"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var s3Client *s3.S3

func getClient() *s3.S3 {
	if s3Client == nil {
		s3Client = s3.New(client.Config())
	}

	return s3Client
}

func BucketExists(bucketName string) bool {
	req := getClient().HeadBucketRequest(&s3.HeadBucketInput{
		Bucket: &bucketName,
	})

	_, err := req.Send()

	return err == nil
}

func CreateBucket(bucketName string) client.Error {
	req := getClient().CreateBucketRequest(&s3.CreateBucketInput{
		Bucket: &bucketName,
	})

	_, err := req.Send()

	return client.NewError(err)
}
