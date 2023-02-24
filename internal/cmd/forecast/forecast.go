// Forecast looks at your account and tries to predict things that will
// go wrong when you attempt to CREATE, UPDATE, or DELETE a stack
package forecast

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/cmd/deploy"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

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
	resource    *yaml.Node
	logicalId   string
	stackExists bool
	stack       types.Stack
	typeName    string
}

// Forecast represents predictions for a single resource in the template
type Forecast struct {
	TypeName  string
	LogicalId string
	Passed    []string
	Failed    []string
}

func (f *Forecast) GetNumChecked() int {
	return len(f.Passed) + len(f.Failed)
}

func (f *Forecast) GetNumFailed() int {
	return len(f.Failed)
}

func (f *Forecast) GetNumPassed() int {
	return len(f.Passed)
}

func (f *Forecast) Append(forecast Forecast) {
	f.Failed = append(f.Failed, forecast.Failed...)
	f.Passed = append(f.Passed, forecast.Passed...)
}

// Add adds a pass or fail message, formatting it to include the type name and logical id
func (f *Forecast) Add(passed bool, message string) {
	// TODO - Add line numbers
	msg := fmt.Sprintf("%v %v - %v", f.TypeName, f.LogicalId, message)
	if passed {
		f.Passed = append(f.Passed, msg)
	} else {
		f.Failed = append(f.Failed, msg)
	}
}

func makeForecast(typeName string, logicalId string) Forecast {
	return Forecast{
		TypeName:  typeName,
		LogicalId: logicalId,
		Passed:    make([]string, 0),
		Failed:    make([]string, 0),
	}
}

// forecasters is a map of resource type names to prediction functions.
var forecasters = make(map[string]func(input PredictionInput) Forecast)

// Push a message about checking a resource onto the spinner
func spin(typeName string, logicalId string, message string) {
	spinner.Push(fmt.Sprintf("%v %v - %v", typeName, logicalId, message))
}

// Run all forecasters for the type
func forecastForType(input PredictionInput) Forecast {

	forecast := makeForecast(input.typeName, input.logicalId)

	// Only run the forecaster if it matches the optional --type arg,
	// or if that arg was not provided.
	if ResourceType != "" && ResourceType != input.typeName {
		config.Debugf("Not running forecasters for %v", input.typeName)
		return forecast
	}

	spin(input.typeName, input.logicalId, "exists already?")

	// Call generic prediction functions that we can run against
	// all resources, even if there is not a predictor.

	// Make sure the resource does not already exist
	if cfn.ResourceAlreadyExists(input.typeName, input.resource, input.stackExists) {
		forecast.Add(false, "Already exists")
	} else {
		forecast.Add(true, "Does not exist")
	}

	// Check permissions
	// (see S3 example, we would need to figure out the arn for each service)
	// TODO - Not sure if this is practical in a generic way

	// See if we have a specific forecaster for this type
	fn, found := forecasters[input.typeName]

	if found {
		// Call the prediction function and append the results
		forecast.Append(fn(input))
	}

	spinner.Pop()

	return forecast
}

// Convert a node to JSON
func toJson(node *yaml.Node) string {
	j, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		return fmt.Sprintf("Failed to marshal node to json: %v:", err)
	}
	return string(j)
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

	forecast := makeForecast("", "")

	rootMap := source.Node.Content[0]

	// Uncomment this to see a json version of the yaml node data model for the template
	// config.Debugf("node: %v", toJson(rootMap))

	// Iterate over each resource
	_, resources := s11n.GetMapValue(rootMap, "Resources")
	if resources == nil {
		panic("Expected to find a Resources section in the template")
	}

	for i, r := range resources.Content {
		if i%2 != 0 {
			continue
		}
		logicalId := r.Value
		config.Debugf("logicalId: %v", logicalId)

		resource := resources.Content[i+1]
		_, typeNode := s11n.GetMapValue(resource, "Type")
		if typeNode == nil {
			panic(fmt.Sprintf("Expected %v to have a Type", logicalId))
		}

		// Check the type and call functions that make checks
		// on that type of resource.

		typeName := typeNode.Value // Should be something like AWS::S3::Bucket
		config.Debugf("typeName: %v", typeName)

		input := PredictionInput{}
		input.logicalId = logicalId
		input.source = source
		input.resource = resource
		input.stackName = stackName
		input.stackExists = stackExists
		input.stack = stack
		input.typeName = typeName

		forecast.Append(forecastForType(input))
	}

	spinner.Stop()

	if forecast.GetNumFailed() > 0 {
		fmt.Println("Stormy weather ahead! 🌪") // 🌩️⛈
		fmt.Println(forecast.GetNumFailed(), "checks failed out of", forecast.GetNumChecked(), "total checks")
		for _, reason := range forecast.Failed {
			fmt.Println()
			fmt.Println(reason)
		}
		return false
	} else {
		fmt.Println("Clear skies! 🌤  All", forecast.GetNumChecked(), "checks passed.")
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
