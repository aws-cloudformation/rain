package s3

import (
	"runtime"

	"github.com/aws-cloudformation/rain/util"
	"github.com/aws-cloudformation/rain/version"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var client *s3.S3

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

	client = s3.New(cfg)
}

func BucketExists(bucketName string) bool {
	req := client.HeadBucketRequest(&s3.HeadBucketInput{
		Bucket: &bucketName,
	})

	_, err := req.Send()

	return err == nil
}

func CreateBucket(bucketName string) error {
	req := client.CreateBucketRequest(&s3.CreateBucketInput{
		Bucket: &bucketName,
	})

	_, err := req.Send()
	return err
}
