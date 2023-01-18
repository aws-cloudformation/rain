package forecast

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
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

// Returns true if the user has the required permissions on the bucket
func checkBucketPermissions(input PredictionInput, bucket *types.StackResourceDetail) bool {

	config.Debugf("Checking if the user has permissions on %v", *bucket.PhysicalResourceId)

	bucketArn := fmt.Sprintf("arn:aws:s3:::%v", *bucket.PhysicalResourceId)
	allAllowed := true

	// Go get the list of permissions from the registry
	actions, err := cfn.GetTypePermissions("AWS::S3::Bucket", "create")
	if err != nil {
		fmt.Println("Unable to get type permissions", err)
		return false
	}
	result, err := iam.Simulate(actions, bucketArn)
	if err != nil {
		return false
	}
	if !result {
		allAllowed = false
	}

	return allAllowed
}

// Check everything that could go wrong with an AWS::S3::Bucket resource
func checkBucket(input PredictionInput) (int, int) {

	res, err := cfn.GetStackResource(input.stackName, input.logicalId)

	if err != nil {
		// If this is an update, the bucket might not exist yet
		config.Debugf("Unable to get details for %v: %v", input.logicalId, err)
		return 0, 0
	}

	bucketName := *res.PhysicalResourceId
	config.Debugf("Physical bucket name is: %v", bucketName)

	// TODO - Put these in a map
	numFailed := 0
	if !checkBucketPermissions(input, res) {
		numFailed += 1
	}
	if !checkBucketNotEmpty(input, res) {
		numFailed += 1
	}
	numChecked := 2
	return numFailed, numChecked

}
