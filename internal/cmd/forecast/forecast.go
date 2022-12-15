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

// An empty bucket cannot be deleted, which will cause a stack DELETE to fail
func checkBucketNotEmpty(input PredictionInput, bucket *types.StackResourceDetail) bool {
	if !input.stackExists {
		return true
	}
	config.Debugf("Checking if the bucket %v is not empty", *bucket.PhysicalResourceId)

	// TODO

	return true
}

func checkBucketPermissions(input PredictionInput, bucket *types.StackResourceDetail) bool {

	config.Debugf("Checking if the user has permissions on %v", *bucket.PhysicalResourceId)

	// TODO

	return false
}

// Check everything that could go wrong with an AWS::S3::Bucket resource
func checkBucket(input PredictionInput) (int, int) {

	res, err := cfn.GetStackResource(input.stackName, input.logicalId)

	if err != nil {
		fmt.Println("Unable to get details for ", input.logicalId, ": ", err)
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

					input := PredictionInput{}
					input.logicalId = logicalId
					input.source = source
					input.resource = resource
					input.stackName = stackName
					input.stackExists = stackExists
					input.stack = stack

					// Check the type and call functions that make checks
					// on that type of resource.

					if elementName == "Type" {

						// See if we have a forecaster for this type
						fn, ok := forecasters[element.(string)]
						if ok {
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
