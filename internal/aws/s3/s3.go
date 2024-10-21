package s3

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/ptr"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/sts"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"

	"github.com/gabriel-vasile/mimetype"
)

var BucketName = ""
var BucketKeyPrefix = ""

func getClient() *s3.Client {
	return s3.NewFromConfig(aws.Config())
}

// BucketHasContents returns true if the bucket is not empty
func BucketHasContents(bucketName string) (bool, error) {

	res, err := getClient().ListObjectVersions(context.Background(),
		&s3.ListObjectVersionsInput{
			Bucket: ptr.String(bucketName),
		})
	if err != nil {
		return false, err
	}
	if res.Versions != nil && len(res.Versions) > 0 {
		return true, nil
	}
	return false, nil
}

// BucketExists checks whether the named bucket exists
func BucketExists(bucketName string) (bool, error) {
	_, err := getClient().HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: ptr.String(bucketName),
	})

	if err != nil {
		var nf *types.NotFound
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateBucket creates a new S3 bucket
func CreateBucket(bucketName string) error {
	input := &s3.CreateBucketInput{
		Bucket: ptr.String(bucketName),
	}

	// We need a location constraint everywhere except us-east-1
	if region := aws.Config().Region; region != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		}
	}

	_, err := getClient().CreateBucket(context.Background(), input)
	if err != nil {
		return err
	}

	// Encrypt the bucket
	_, err = getClient().PutBucketEncryption(context.Background(), &s3.PutBucketEncryptionInput{
		Bucket: ptr.String(bucketName),
		ServerSideEncryptionConfiguration: &types.ServerSideEncryptionConfiguration{
			Rules: []types.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
						SSEAlgorithm: types.ServerSideEncryptionAes256,
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	// Add public access block
	_, err = getClient().PutPublicAccessBlock(context.Background(), &s3.PutPublicAccessBlockInput{
		Bucket: ptr.String(bucketName),
		PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       awssdk.Bool(true),
			BlockPublicPolicy:     awssdk.Bool(true),
			IgnorePublicAcls:      awssdk.Bool(true),
			RestrictPublicBuckets: awssdk.Bool(true),
		},
	})
	if err != nil {
		return err
	}

	// Add lifecycle config
	_, err = getClient().PutBucketLifecycleConfiguration(context.Background(), &s3.PutBucketLifecycleConfigurationInput{
		Bucket: ptr.String(bucketName),
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{
			Rules: []types.LifecycleRule{
				{
					Status: types.ExpirationStatusEnabled,
					AbortIncompleteMultipartUpload: &types.AbortIncompleteMultipartUpload{
						DaysAfterInitiation: awssdk.Int32(7),
					},
					Expiration: &types.LifecycleExpiration{
						Days: awssdk.Int32(7),
					},
					Filter: &types.LifecycleRuleFilter{
						Prefix: awssdk.String(""),
					},
					ID: ptr.String("delete after 14 days"),
					NoncurrentVersionExpiration: &types.NoncurrentVersionExpiration{
						NoncurrentDays: awssdk.Int32(7),
					},
				},
			},
		},
	})

	return err
}

// Upload uploads an artifact to the bucket with a unique name
func Upload(bucketName string, content []byte) (string, error) {
	isBucketExists, errBucketExists := BucketExists(bucketName)

	if errBucketExists != nil {
		return "", fmt.Errorf("unable to confirm whether artifact bucket exists: %w", errBucketExists)
	}

	if !isBucketExists {
		return "", fmt.Errorf("bucket does not exist: '%s'", bucketName)
	}

	key := filepath.Join(BucketKeyPrefix, fmt.Sprintf("%x", sha256.Sum256(content)))

	_, err := getClient().PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: ptr.String(bucketName),
		Key:    ptr.String(key),
		Body:   bytes.NewReader(content),
	})

	config.Debugf("Artifact key: %s", key)

	return key, err
}

// RainBucket returns the name of the rain deployment bucket in the current region
// and asks the user if they wish it to be created if it does not exist
// unless forceCreation is true, then it will not ask
func RainBucket(forceCreation bool) string {
	accountID, err := sts.GetAccountID()
	if err != nil {
		panic(fmt.Errorf("unable to get account ID: %w", err))
	}

	bucketName := BucketName
	if bucketName == "" {
		bucketName = fmt.Sprintf("rain-artifacts-%s-%s", accountID, aws.Config().Region)
	}

	config.Debugf("Artifact bucket: %s", bucketName)

	isBucketExists, err := BucketExists(bucketName)
	if err != nil {
		panic(fmt.Errorf("unable to confirm whether artifact bucket exists: %w", err))
	}

	if !isBucketExists {
		spinner.Pause()
		if !forceCreation && !console.Confirm(true, fmt.Sprintf("Rain needs to create an S3 bucket called '%s'. Continue?", bucketName)) {
			panic(errors.New("you may create the bucket manually and then re-run this operation"))
		}
		spinner.Resume()

		err := CreateBucket(bucketName)
		if err != nil {
			panic(fmt.Errorf("unable to create artifact bucket '%s': %w", bucketName, err))
		}
	}

	// Sleep for 2 seconds to give the bucket time to stabilize
	time.Sleep(2 * time.Second)

	// #213
	// Confirm that the bucket really does exist.
	// Seems unnecessary but bug 213 looks like a race condition. Maybe
	// checking here and pausing a few seconds will be enough?
	isBucketExists, err = BucketExists(bucketName)
	if err != nil {
		config.Debugf("unable to confirm bucket after creation: %v", err)
	}

	if !isBucketExists {
		// Sleep for 5 seconds
		time.Sleep(5 * time.Second)

		// Check again
		isBucketExists, err = BucketExists(bucketName)
		if err != nil {
			panic(fmt.Errorf("unable to re-confirm whether artifact bucket exists: %w", err))
		}

		// Give up
		if !isBucketExists {
			panic(fmt.Errorf("cannot confirm that artifact bucket '%s' exists", bucketName))
		}
	}

	return bucketName
}

// GetObject gets an object by key from an S3 bucket
func GetObject(bucketName string, key string) ([]byte, error) {
	result, err := getClient().GetObject(context.Background(),
		&s3.GetObjectInput{
			Bucket: &bucketName,
			Key:    &key,
		})
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// GetUnzippedObjectSize gets the uncompressed length in bytes of an object.
// Calling this on a large object will be slow!
func GetUnzippedObjectSize(bucketName string, key string) (int64, error) {
	result, err := getClient().GetObject(context.Background(),
		&s3.GetObjectInput{
			Bucket: &bucketName,
			Key:    &key,
		})
	if err != nil {
		return 0, err
	}
	var size int64 = 0

	body, err := io.ReadAll(result.Body)
	if err != nil {
		return 0, err
	}

	// Unzip the archive and count the total bytes of all files
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		// TODO: What if it's not a zip file? Maybe return something like -1?
		return 0, err
	}

	// Read all the files from zip archive and count total size
	for _, zipFile := range zipReader.File {
		config.Debugf("Reading file from zip archive: %s", zipFile.Name)

		f, err := zipFile.Open()
		if err != nil {
			config.Debugf("Error opening zip file %s: %v", zipFile.Name, err)
			return 0, err
		}
		defer f.Close()

		bytesRead := 0
		buf := make([]byte, 256)
		for {
			bytesRead, err = f.Read(buf)
			if err != nil {
				config.Debugf("Error reading from zip file %s: %v", zipFile.Name, err)
			}
			if bytesRead == 0 {
				break
			}
			size += int64(bytesRead)
		}
	}

	config.Debugf("Total size for %s/%s is %d", bucketName, key, size)

	return size, nil
}

type S3ObjectInfo struct {
	SizeBytes int64
}

// HeadObject gets information about an object without downloading it
func HeadObject(bucketName string, key string) (*S3ObjectInfo, error) {
	result, err := getClient().HeadObject(context.Background(),
		&s3.HeadObjectInput{
			Bucket: &bucketName,
			Key:    &key,
		})
	if err != nil {
		return nil, err
	}
	retval := &S3ObjectInfo{
		SizeBytes: *result.ContentLength,
	}
	return retval, nil
}

// PutObject puts an object into a bucket
func PutObject(bucketName string, key string, body []byte) error {

	// Determine the correct content type
	// This seems to be the default. It breaks web pages served by S3.
	contentType := "application/octet-stream"

	mtype := mimetype.Detect(body)
	contentType = mtype.String()
	config.Debugf("PutObject determine mime type for %s: %s", key, contentType)

	if strings.HasSuffix(key, ".css") {
		config.Debugf("PutObject changing css content type to text/css")
		contentType = strings.Replace(contentType, "text/plain", "text/css", 1)
	}

	if strings.HasSuffix(key, ".js") {
		config.Debugf("PutObject changing js content type to text/js")
		contentType = strings.Replace(contentType, "text/plain", "text/javascript", 1)
	}

	config.Debugf("PutObject final mime type for %s: %s", key, contentType)

	_, err := getClient().PutObject(context.Background(),
		&s3.PutObjectInput{
			Bucket:      &bucketName,
			Key:         &key,
			Body:        bytes.NewReader(body),
			ContentType: &contentType,
		})
	return err
}

// DeleteObject deletes an object from a bucket
func DeleteObject(bucketName string, key string, version *string) error {
	_, err := getClient().DeleteObject(context.Background(),
		&s3.DeleteObjectInput{
			Bucket:    &bucketName,
			Key:       &key,
			VersionId: version,
		})
	return err
}

// EmptyBucket deletes all versions of all objects in a bucket
func EmptyBucket(bucketName string) error {
	client := getClient()
	input := &s3.ListObjectsV2Input{
		Bucket: &bucketName,
	}
	for {
		res, err := client.ListObjectsV2(context.Background(), input)
		if err != nil {
			return err
		}
		for _, item := range res.Contents {
			config.Debugf("Deleting %s/%s", bucketName, *item.Key)
			derr := DeleteObject(bucketName, *item.Key, nil)
			if derr != nil {
				return derr
			}
		}
		if *res.IsTruncated {
			input.ContinuationToken = res.ContinuationToken
		} else {
			break
		}
	}

	// Now delete old versions and delete markers

	vinput := &s3.ListObjectVersionsInput{
		Bucket: &bucketName,
	}
	for {
		res, err := client.ListObjectVersions(context.Background(), vinput)
		if err != nil {
			return err
		}

		for _, item := range res.DeleteMarkers {
			config.Debugf("Deleting delete marker %s/%s v%s", bucketName, *item.Key, *item.VersionId)
			DeleteObject(bucketName, *item.Key, item.VersionId)
		}

		for _, item := range res.Versions {
			config.Debugf("Deleting version %s/%s v%s", bucketName, *item.Key, *item.VersionId)
			DeleteObject(bucketName, *item.Key, item.VersionId)
		}

		if *res.IsTruncated {
			vinput.VersionIdMarker = res.NextVersionIdMarker
			vinput.KeyMarker = res.NextKeyMarker
		} else {
			break
		}
	}

	return nil
}
