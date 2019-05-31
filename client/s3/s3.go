package s3

import (
	"runtime"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws-cloudformation/rain/util"
	"github.com/aws-cloudformation/rain/version"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var s3Client *s3.S3

func init() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		util.Die(err)
	}

	// Set the user agent
	cfg.Handlers.Build.Remove(defaults.SDKVersionUserAgentHandler)
	cfg.Handlers.Build.PushFront(aws.MakeAddToUserAgentHandler(
		version.NAME,
		version.VERSION,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	))

	s3Client = s3.New(cfg)
}

func BucketExists(bucketName string) bool {
	req := s3Client.HeadBucketRequest(&s3.HeadBucketInput{
		Bucket: &bucketName,
	})

	_, err := req.Send()

	return err == nil
}

func CreateBucket(bucketName string) client.Error {
	req := s3Client.CreateBucketRequest(&s3.CreateBucketInput{
		Bucket: &bucketName,
	})

	_, err := req.Send()

	return client.NewError(err)
}
