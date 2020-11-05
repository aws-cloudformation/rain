//+build func_test

package s3

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/google/uuid"
)

var buckets = make(map[string]bool)

// BucketExists checks whether the named bucket exists
func BucketExists(bucketName string) bool {
	_, ok := buckets[bucketName]
	return ok
}

// CreateBucket creates a new S3 bucket
func CreateBucket(bucketName string) error {
	buckets[bucketName] = true
	return nil
}

// Upload an artefact to the bucket with a unique name
func Upload(bucketName, content string) (string, error) {
	if !BucketExists(bucketName) {
		return "", fmt.Errorf("Bucket does not exist: '%s'", bucketName)
	}

	return uuid.New().String(), nil
}

// RainBucket returns the name of the rain deployment bucket in the current region
// and creates it if it does not exist
func RainBucket() string {
	bucketName := fmt.Sprintf("rain-artifacts-1234567890-%s", aws.Config().Region)

	config.Debugf("Artifact bucket: %s", bucketName)

	if !BucketExists(bucketName) {
		config.Debugf("Mock creating rain bucket '%s'", bucketName)

		err := CreateBucket(bucketName)
		if err != nil {
			panic(fmt.Errorf("unable to create artifact bucket '%s': %w", bucketName, err))
		}
	}

	return bucketName
}
