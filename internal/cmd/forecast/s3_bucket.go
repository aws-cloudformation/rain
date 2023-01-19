package forecast

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	"github.com/google/uuid"
)

// An empty bucket cannot be deleted, which will cause a stack DELETE to fail.
// Returns true if the stack operation will succeed.
func checkBucketNotEmpty(input PredictionInput, bucket *types.StackResourceDetail) bool {
	if !input.stackExists {
		return true
	}
	config.Debugf("Checking if the bucket %v is not empty", *bucket.PhysicalResourceId)

	exists, err := s3.BucketExists(*bucket.PhysicalResourceId)
	if err != nil || !exists {
		// The bucket might not exist if this is an UPDATE with new resources
		// But we should have already handled this when we got resource details
		fmt.Println(*bucket.LogicalResourceId, "does not exist.", err)
		return false
	}

	hasContents, _ := s3.BucketHasContents(*bucket.PhysicalResourceId)
	if hasContents {
		// Check the deletion policy
		for elementName, element := range input.resource.(map[string]interface{}) {
			config.Debugf("checkBucketNotEmpty element %v %v", elementName, element)
			if elementName == "DeletionPolicy" {
				if element == "Retain" {
					// The bucket is not empty but it is set to retain,
					// so a stack DELETE will not fail
					return true
				}
			}
		}
		fmt.Println(*bucket.LogicalResourceId, "is not empty, so a stack DELETE will fail")
	}
	return !hasContents
}

// Check everything that could go wrong with an AWS::S3::Bucket resource
func checkBucket(input PredictionInput) (int, int) {

	// A uuid will be used for policy silumation if the bucket does not already exist
	bucketName := fmt.Sprintf("rain-%v", uuid.New())
	bucketArn := ""
	numFailed := 0
	numChecked := 0

	if input.stackExists {
		res, err := cfn.GetStackResource(input.stackName, input.logicalId)

		if err != nil {
			// If this is an update, the bucket might not exist yet
			config.Debugf("Unable to get details for %v: %v", input.logicalId, err)
		} else {
			// The bucket exists
			bucketName := *res.PhysicalResourceId
			config.Debugf("Physical bucket name is: %v", bucketName)

			if !checkBucketNotEmpty(input, res) {
				numFailed += 1
				numChecked += 1
			}
		}
	}

	bucketArn = fmt.Sprintf("arn:aws:s3:::%v", bucketName)

	// TODO - Can we make the permissions check generic so we can
	// run it on all types? What if we can't predict what the arn will be?
	if input.stackExists {
		if !checkPermissions(input, bucketArn, "update") {
			fmt.Println("Insufficient permissions to update", bucketArn)
			numFailed += 1
		}
		numChecked += 1
		if !checkPermissions(input, bucketArn, "delete") {
			fmt.Println("Insufficient permissions to delete", bucketArn)
			numFailed += 1
		}
		numChecked += 1
	} else {
		if !checkPermissions(input, bucketArn, "create") {
			fmt.Println("Insufficient permissions to create", bucketArn)
			numFailed += 1
		}
		numChecked += 1
	}

	return numFailed, numChecked

}
