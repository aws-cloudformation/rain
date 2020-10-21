package s3

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws-cloudformation/rain/client/sts"
	"github.com/aws-cloudformation/rain/config"
	"github.com/aws-cloudformation/rain/console"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

func getClient() *s3.Client {
	return s3.NewFromConfig(client.Config())
}

// BucketExists checks whether the named bucket exists
func BucketExists(bucketName string) bool {
	_, err := getClient().HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})

	return err == nil
}

// CreateBucket creates a new S3 bucket
func CreateBucket(bucketName string) error {
	_, err := getClient().CreateBucket(context.Background(), &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
		ACL:    types.BucketCannedACLPrivate,
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(client.Config().Region),
		},
	})

	return err
}

// Upload an artefact to the bucket with a unique name
func Upload(bucketName, content string) (string, error) {
	if !BucketExists(bucketName) {
		return "", fmt.Errorf("Bucket does not exist: '%s'", bucketName)
	}

	key := uuid.New().String()

	_, err := getClient().PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
		ACL:    types.ObjectCannedACLPrivate,
		Body:   strings.NewReader(content),
	})

	config.Debugf("Artifact key: %s", key)

	return key, err
}

// RainBucket returns the name of the rain deployment bucket in the current region
// and creates it if it does not exist
func RainBucket() string {
	accountID, err := sts.GetAccountID()
	if err != nil {
		panic(fmt.Errorf("unable to get account ID: %w", err))
	}

	bucketName := fmt.Sprintf("rain-artifacts-%s-%s", accountID, client.Config().Region)

	config.Debugf("Artifact bucket: %s", bucketName)

	if !BucketExists(bucketName) {
		if !console.Confirm(true, fmt.Sprintf("Rain needs to create an S3 bucket called '%s'. Continue?", bucketName)) {
			panic(errors.New("you may create the bucket manually and then re-run this operation"))
		}

		err := CreateBucket(bucketName)
		if err != nil {
			panic(fmt.Errorf("unable to create artifact bucket '%s': %w", bucketName, err))
		}
	}

	return bucketName
}
