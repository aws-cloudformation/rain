package forecast

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/s11n"
	fc "github.com/aws-cloudformation/rain/plugins/forecast"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// An empty bucket cannot be deleted, which will cause a stack DELETE to fail.
// Returns true if the stack operation will succeed.
func checkBucketNotEmpty(input fc.PredictionInput, bucket *types.StackResourceDetail) (bool, string) {
	if !input.StackExists {
		return true, "Stack does not exist"
	}

	if bucket == nil {
		return false, "bucket is nil"
	}

	spin(input.TypeName, input.LogicalId, "bucket not empty?")

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
		_, deletionPolicy, _ := s11n.GetMapValue(input.Resource, "DeletionPolicy")
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
func CheckS3Bucket(input fc.PredictionInput) fc.Forecast {

	forecast := fc.MakeForecast(&input)

	if input.StackExists {
		res, err := cfn.GetStackResource(input.StackName, input.LogicalId)

		if err != nil {
			// If this is an update, the bucket might not exist yet
			config.Debugf("Unable to get details for %v: %v", input.LogicalId, err)
		} else {
			// The bucket exists
			bucketName := *res.PhysicalResourceId
			config.Debugf("Physical bucket name is: %v", bucketName)

			empty, reason := checkBucketNotEmpty(input, res)
			code := F0001
			if !empty {
				forecast.Add(code, false, reason, getLineNum(input.LogicalId, input.Resource))
			} else {
				forecast.Add(code, true, "Bucket is empty", getLineNum(input.LogicalId, input.Resource))
			}
		}
	} else {
		config.Debugf("Stack does not exist, not checking if bucket is empty")
	}

	return forecast
}
