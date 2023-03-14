package forecast

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	"github.com/google/uuid"
)

// An empty bucket cannot be deleted, which will cause a stack DELETE to fail.
// Returns true if the stack operation will succeed.
func checkBucketNotEmpty(input PredictionInput, bucket *types.StackResourceDetail) (bool, string) {
	if !input.stackExists {
		return true, "Stack does not exist"
	}

	spin(input.typeName, input.logicalId, "bucket not empty?")

	config.Debugf("Checking if the bucket %v is not empty", *bucket.PhysicalResourceId)

	exists, err := s3.BucketExists(*bucket.PhysicalResourceId)
	if err != nil {
		return false, fmt.Sprintf("Unable to check if bucket exists: %v", err)
	}

	if !exists {
		// The bucket might not exist if this is an UPDATE with new resources
		// But we should have already handled this when we got resource details
		return false, "Bucket does not exist"
	}

	hasContents, _ := s3.BucketHasContents(*bucket.PhysicalResourceId)
	if hasContents {
		// Check the deletion policy
		_, deletionPolicy := s11n.GetMapValue(input.resource, "DeletionPolicy")
		if deletionPolicy != nil && deletionPolicy.Value == "Retain" {
			// The bucket is not empty but it is set to retain,
			// so a stack DELETE will not fail
			return true, "Bucket is not empty but is set to RETAIN"
		}
		return false, "Bucket is not empty, so a stack DELETE will fail"

		// TODO - Should we check to see if they are using something like
		// AwsCommunity::S3::DeleteBucketContents?
		// (or a similar custom resource? .. not sure how to do this reliably)
	}

	spinner.Pop()

	return true, ""
}

// Check everything that could go wrong with an AWS::S3::Bucket resource
func checkS3Bucket(input PredictionInput) Forecast {

	// A uuid will be used for policy silumation if the bucket does not already exist
	bucketName := fmt.Sprintf("rain-%v", uuid.New())
	bucketArn := ""

	forecast := makeForecast(input.typeName, input.logicalId)

	if input.stackExists {
		res, err := cfn.GetStackResource(input.stackName, input.logicalId)

		if err != nil {
			// If this is an update, the bucket might not exist yet
			config.Debugf("Unable to get details for %v: %v", input.logicalId, err)
		} else {
			// The bucket exists
			bucketName := *res.PhysicalResourceId
			config.Debugf("Physical bucket name is: %v", bucketName)

			empty, reason := checkBucketNotEmpty(input, res)
			if !empty {
				forecast.Add(false, reason)
			} else {
				forecast.Add(true, "Bucket is empty")
			}
		}
	}

	bucketArn = fmt.Sprintf("arn:aws:s3:::%v", bucketName)

	// TODO - Can we make the permissions check generic so we can
	// run it on all types? What if we can't predict what the arn will be?
	// We could have a map of resource names to functions that provide the arn..
	var ok bool
	var reason []string
	if input.stackExists {
		ok, reason = checkPermissions(input, bucketArn, "update")
		if !ok {
			forecast.Add(false, fmt.Sprintf("Insufficient permissions to update %v\n\t%v", bucketArn, strings.Join(reason, "\n\t")))
		} else {
			forecast.Add(true, "Role has update permissions")
		}

		ok, reason = checkPermissions(input, bucketArn, "delete")
		if !ok {
			forecast.Add(false, fmt.Sprintf("Insufficient permissions to delete %v\n\t%v", bucketArn, strings.Join(reason, "\n\t")))
		} else {
			forecast.Add(true, "Role has delete permissions")
		}
	} else {
		ok, reason = checkPermissions(input, bucketArn, "create")
		if !ok {
			forecast.Add(false, fmt.Sprintf("Insufficient permissions to create %v\n\t%v", bucketArn, strings.Join(reason, "\n\t")))
		} else {
			forecast.Add(true, "Role has create permissions")
		}
	}

	return forecast
}
