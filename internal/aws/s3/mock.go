//go:build func_test

package s3

import (
	"crypto/sha256"
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
)

var buckets = make(map[string]bool)

// BucketExists checks whether the named bucket exists
func BucketExists(bucketName string) (bool, error) {
	_, ok := buckets[bucketName]
	return ok, nil
}

// CreateBucket creates a new S3 bucket
func CreateBucket(bucketName string) error {
	buckets[bucketName] = true
	return nil
}

// Upload an artefact to the bucket with a unique name
func Upload(bucketName string, content []byte) (string, error) {
	isBucketExists, _ := BucketExists(bucketName)
	if !isBucketExists {
		return "", fmt.Errorf("bucket does not exist: '%s'", bucketName)
	}

	return fmt.Sprintf("%x", sha256.Sum256(content)), nil
}

// RainBucket returns the name of the rain deployment bucket in the current region
// and creates it if it does not exist
func RainBucket(forceCreation bool) string {
	bucketName := fmt.Sprintf("rain-artifacts-1234567890-%s", aws.Config().Region)

	config.Debugf("Artifact bucket: %s", bucketName)

	isBucketExists, _ := BucketExists(bucketName)
	if !isBucketExists {
		if forceCreation {
			config.Debugf("Force creating rain bucket '%s'", bucketName)
		} else {
			config.Debugf("Mock creating rain bucket '%s'", bucketName)
		}

		err := CreateBucket(bucketName)
		if err != nil {
			panic(fmt.Errorf("unable to create artifact bucket '%s': %w", bucketName, err))
		}
	}

	return bucketName
}
