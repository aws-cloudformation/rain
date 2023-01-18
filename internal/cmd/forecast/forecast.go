// Forecast looks at your account and tries to predict things that will
// go wrong when you attempt to CREATE, UPDATE, or DELETE a stack
package forecast

import (
	"fmt"
	"io"
	"os"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/cmd/deploy"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/spf13/cobra"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// Input to forecast prediction functions
type PredictionInput struct {
	source      cft.Template
	stackName   string
	resource    interface{}
	logicalId   string
	stackExists bool
	stack       types.Stack
}

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

	// TODO
	// https://github.com/aws/aws-sdk-go-v2/blob/main/service/iam/api_op_SimulatePrincipalPolicy.go

	bucketArn := fmt.Sprintf("arn:aws:s3:::%v", bucket.PhysicalResourceId)
	allAllowed := true

	result, err := iam.Simulate("s3:CreateBucket", bucketArn)
	if err != nil {
		return false
	}
	if !result {
		allAllowed = false
	}

	result, err = iam.Simulate("s3:DeleteBucket", bucketArn)
	if err != nil {
		return false
	}
	if !result {
		allAllowed = false
	}

	// TODO - Should we go get the list of permissions from the registry?
	// (retrieve programatically or just hard code them here?)
	//
	// for example:
	//
	/* "handlers": {
	   "create": {
	       "permissions": [
	           "s3:CreateBucket",
	           "s3:PutBucketTagging",
	           "s3:PutAnalyticsConfiguration",
	           "s3:PutEncryptionConfiguration",
	           "s3:PutBucketCORS",
	           "s3:PutInventoryConfiguration",
	           "s3:PutLifecycleConfiguration",
	           "s3:PutMetricsConfiguration",
	           "s3:PutBucketNotification",
	           "s3:PutBucketReplication",
	           "s3:PutBucketWebsite",
	           "s3:PutAccelerateConfiguration",
	           "s3:PutBucketPublicAccessBlock",
	           "s3:PutReplicationConfiguration",
	           "s3:PutObjectAcl",
	           "s3:PutBucketObjectLockConfiguration",
	           "s3:GetBucketAcl",
	           "s3:ListBucket",
	           "iam:PassRole",
	           "s3:DeleteObject",
	           "s3:PutBucketLogging",
	           "s3:PutBucketVersioning",
	           "s3:PutObjectLockConfiguration",
	           "s3:PutBucketOwnershipControls",
	           "s3:PutBucketIntelligentTieringConfiguration"
	*/

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

// Query the account to make predictions about deployment failures
func predict(source cft.Template, stackName string) {

	config.Debugf("About to make API calls for failure prediction...")

	// Visit each resource in the template and see if it matches
	// one of our predictions

	// First check to see if the stack already exists.
	// If so, check for possible update issues, and for reasons we can't delete the stack
	// Otherwise, only check for possible create failures
	stack, stackExists := deploy.CheckStack(stackName)

	msg := ""
	if stackExists {
		msg = "exists"
	} else {
		msg = "does not exist"
	}
	config.Debugf("Stack %v %v", stackName, msg)

	numChecked := 0
	numFailed := 0

	// A map of resource type names to prediction functions
	// The function returns (numFailed, numChecked)
	forecasters := make(map[string]func(input PredictionInput) (int, int))

	// If you want to add a prediction for a type that is not already covered, add it here
	// The function must return (numFailed, numChecked)
	// For example:
	// forecasters["AWS::New::Type"] = checkTheNewType

	forecasters["AWS::S3::Bucket"] = checkBucket

	m := source.Map()
	for t, section := range m {
		if t == "Resources" {
			// Iterate over each resource
			for logicalId, resource := range section.(map[string]interface{}) {
				config.Debugf("resource %v %v", logicalId, resource)
				// Iterate over each element in the resource
				for elementName, element := range resource.(map[string]interface{}) {
					config.Debugf("element %v %v", elementName, element)

					// Check the type and call functions that make checks
					// on that type of resource.

					if elementName == "Type" {

						// See if we have a forecaster for this type
						fn, ok := forecasters[element.(string)]
						if ok {

							input := PredictionInput{}
							input.logicalId = logicalId
							input.source = source
							input.resource = resource
							input.stackName = stackName
							input.stackExists = stackExists
							input.stack = stack

							// Call the prediction function
							nf, nc := fn(input)
							numFailed += nf
							numChecked += nc
						}
					}
				}
			}
		}
	}

	if numFailed > 0 {
		fmt.Println("Stormy weather ahead!")
		fmt.Println(numFailed, "checks failed out of", numChecked, "total checks")
	} else {
		fmt.Println("Clear skies! All", numChecked, "checks passed.")
	}

	// TODO - We might be able to incorporate AWS Config proactive controls here
	// https://aws.amazon.com/blogs/aws/new-aws-config-rules-now-support-proactive-compliance/

	// What about hooks? Could we invoke those handlers to see if they will fail before deployment?

}

// Cmd is the forecast command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "forecast <template> <stackName>",
	Short:                 "Predict deployment failures",
	Long:                  "Outputs warnings about potential deployment failures due to constraints in the account or misconfigurations in the template related to dependencies in the account.",
	Args:                  cobra.ExactArgs(2),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fn := args[0]
		stackName := args[1]

		// ? Should we include the command (create/update/delete) in the args?
		// ? Should this run on a change set?
		// TODO - Only look at the diff for updates

		config.Debugf("Generating forecast...", fn)

		r, err := os.Open(fn)
		if err != nil {
			panic(ui.Errorf(err, "unable to read '%s'", fn))
		}

		// Read the template
		input, err := io.ReadAll(r)
		if err != nil {
			panic(ui.Errorf(err, "unable to read input"))
		}

		// Parse the template
		source, err := parse.String(string(input))
		if err != nil {
			panic(ui.Errorf(err, "unable to parse input"))
		}

		predict(source, stackName)

	},
}

func init() {
	Cmd.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
}
