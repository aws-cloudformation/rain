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
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/spf13/cobra"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// The role name to use for the IAM policy simulator (optional --role)
var RoleArn string

// The resource type to check (optional --type to limit checks to one type)
var ResourceType string

// The optional parameters to use to create a change set for update predictions (--params)
var Params []string

// The optional path to a file that contains params (--config)
var ConfigFilePath string

// Input to forecast prediction functions
type PredictionInput struct {
	source      cft.Template
	stackName   string
	resource    interface{}
	logicalId   string
	stackExists bool
	stack       types.Stack
	typeName    string
}

// Forecast represents predictions for a single resource in the template
type Forecast struct {
	ResourceType string
	LogicalId    string
	Passed       []string
	Failed       []string
}

func (f Forecast) GetNumChecked() int {
	return len(f.Passed) + len(f.Failed)
}

func (f Forecast) GetNumFailed() int {
	return len(f.Failed)
}

func (f Forecast) GetNumPassed() int {
	return len(f.Passed)
}

func (f Forecast) Append(forecast Forecast) {
	f.Failed = append(f.Failed, forecast.Failed...)
	f.Passed = append(f.Failed, forecast.Passed...)
}

func (f Forecast) Add(passed bool, message string) {
	if passed {
		f.Passed = append(f.Passed, message)
	} else {
		f.Failed = append(f.Failed, message)
	}
}

func makeForecast(typeName string, logicalId string) Forecast {
	return Forecast{
		ResourceType: typeName,
		LogicalId:    logicalId,
		Passed:       make([]string, 0),
		Failed:       make([]string, 0),
	}
}

// forecasters is a map of resource type names to prediction functions.
var forecasters = make(map[string]func(input PredictionInput) Forecast)

// Run all forecasters for the type
func forecastForType(typeName string, input PredictionInput) Forecast {

	forecast := makeForecast(typeName, input.logicalId)

	// Only run the forecaster if it matches the optional --type arg,
	// or if that arg was not provided.
	if ResourceType != "" && ResourceType != typeName {
		config.Debugf("Not running forecasters for %v", typeName)
		return forecast
	}

	spinner.Push(fmt.Sprintf("Checking %v %v", input.logicalId, typeName))

	// Call generic prediction functions that we can run against
	// all resources, even if there is not a predictor.

	// Make sure the resource does not already exist
	if cfn.ResourceAlreadyExists(typeName, input.resource.(map[string]interface{}), input.stackExists) {
		forecast.Add(false, "Already exists")
	} else {
		forecast.Add(true, "Does not exist")
	}

	// Check permissions
	// (see S3 example, we would need to figure out the arn for each service)
	// TODO - Not sure if this is practical in a generic way

	// See if we have a specific forecaster for this type
	fn, ok := forecasters[typeName]

	if ok {
		// Call the prediction function and append the results
		forecast.Append(fn(input))
	}

	return forecast
}

// Query the account to make predictions about deployment failures.
// Returns true if no failures are predicted.
func predict(source cft.Template, stackName string) bool {

	config.Debugf("About to make API calls for failure prediction...")

	// Visit each resource in the template and see if it matches
	// one of our predictions

	// First check to see if the stack already exists.
	// If so, check for possible update issues, and for reasons we can't delete the stack
	// Otherwise, only check for possible create failures
	stack, stackExists := deploy.CheckStack(stackName)

	// TODO: Add all the same params as the `deploy` command has so we can create
	// a change set for updates.

	msg := ""
	if stackExists {
		msg = "exists"
	} else {
		msg = "does not exist"
	}
	config.Debugf("Stack %v %v", stackName, msg)

	numChecked := 0
	numFailed := 0

	m := source.Map()
	resources := m["Resources"]

	// Iterate over each resource
	for logicalId, resource := range resources.(map[string]interface{}) {
		config.Debugf("resource %v %v", logicalId, resource)

		t := resource.(map[string]interface{})["Type"]

		// Check the type and call functions that make checks
		// on that type of resource.

		typeName := t.(string) // Should be something like AWS::S3::Bucket
		config.Debugf("typeName: %v", typeName)

		input := PredictionInput{}
		input.logicalId = logicalId
		input.source = source
		input.resource = resource
		input.stackName = stackName
		input.stackExists = stackExists
		input.stack = stack
		input.typeName = typeName

		forecast := forecastForType(typeName, input)
		numFailed += forecast.GetNumFailed()
		numChecked += forecast.GetNumChecked()
	}

	if numFailed > 0 {
		fmt.Println("Stormy weather ahead!")
		fmt.Println(numFailed, "checks failed out of", numChecked, "total checks")
		return false
	} else {
		fmt.Println("Clear skies! All", numChecked, "checks passed.")
		return true
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

		config.Debugf("Generating forecast for %v", fn)

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

		if !predict(source, stackName) {
			os.Exit(1)
		}

	},
}

func init() {
	Cmd.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
	Cmd.Flags().StringVar(&RoleArn, "role-arn", "", "An optional execution role arn to use for predicting IAM failures")
	// TODO - --op "create", "update", "delete", default: "all"
	Cmd.Flags().StringVar(&ResourceType, "type", "", "Optional resource type to limit checks to only that type")
	Cmd.Flags().StringSliceVar(&Params, "params", []string{}, "set parameter values; use the format key1=value1,key2=value2")
	Cmd.Flags().StringVarP(&ConfigFilePath, "config", "c", "", "YAML or JSON file to set tags and parameters")

	// If you want to add a prediction for a type that is not already covered, add it here
	// The function must return (numFailed, numChecked)
	// For example:
	// forecasters["AWS::New::Type"] = checkTheNewType

	forecasters["AWS::S3::Bucket"] = checkBucket
	forecasters["AWS::S3::BucketPolicy"] = checkBucketPolicy

}
