// Package forecast looks at your account and tries to predict things that will
// go wrong when you attempt to CREATE, UPDATE, or DELETE a stack
package forecast

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/iam"
	"github.com/aws-cloudformation/rain/internal/cmd/deploy"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/dc"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// RoleArn is the role name to use for the IAM policy simulator (optional --role)
var RoleArn string

// Experimental indicates that this is an experimental feature that might break between minor releases
var Experimental bool

// ResourceType is the resource type to check (optional --type to limit checks to one type)
var ResourceType string

// IncludeIAM indicates if we should perform permissions checks or not, to save time
var IncludeIAM bool

// The optional parameters to use to create a change set for update predictions (--params)
var params []string

// The optional tags to use to create a change set for update predictions (--tags)
var tags []string

// The optional path to a file that contains params (--config)
var configFilePath string

// Show success in addition to failures
var all bool

// Which stack action to check (--action)
var action string

// Which checks to ignore (--ignore)
var ignore []string

const (
	ALL    = "all"
	CREATE = "create"
	UPDATE = "update"
	DELETE = "delete"
)

type Env struct {
	partition string
	region    string
	account   string
}

// PredictionInput is the input to forecast prediction functions
type PredictionInput struct {
	source      cft.Template
	stackName   string
	resource    *yaml.Node
	logicalId   string
	stackExists bool
	stack       types.Stack
	typeName    string
	dc          *dc.DeployConfig
	env         Env
	roleArn     string
}

// GetPropertyNode returns the node for the given property name
func (input *PredictionInput) GetPropertyNode(name string) *yaml.Node {
	_, props, _ := s11n.GetMapValue(input.resource, "Properties")
	if props != nil {
		_, n, _ := s11n.GetMapValue(props, name)
		return n
	}
	return nil
}

// LineNumber is the current line number in the template
var LineNumber int

// Forecast represents predictions for a single resource in the template
type Forecast struct {
	TypeName  string
	LogicalId string
	Passed    []Check
	Failed    []Check
}

// Check is a specific check with a code that can be suppressed
type Check struct {
	Pass    bool
	Code    string
	Message string
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
func (f *Forecast) Add(code string, passed bool, message string) {
	msg := fmt.Sprintf("%v: %v %v - %v", LineNumber, f.TypeName, f.LogicalId, message)
	check := Check{
		Pass:    passed,
		Code:    code,
		Message: msg,
	}

	// If we are ignoring this check, don't add it to the forecast
	if slices.Contains(ignore, code) || slices.Contains(ignore, f.TypeName) {
		return
	}

	if passed {
		f.Passed = append(f.Passed, check)
	} else {
		f.Failed = append(f.Failed, check)
	}
}

func (c *Check) String() string {
	var passFail string
	if c.Pass {
		passFail = "PASS"
	} else {
		passFail = "FAIL"
	}
	return fmt.Sprintf("%s %s on line %s", c.Code, passFail, c.Message)
}

func makeForecast(typeName string, logicalId string) Forecast {
	return Forecast{
		TypeName:  typeName,
		LogicalId: logicalId,
		Passed:    make([]Check, 0),
		Failed:    make([]Check, 0),
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

	// Resolve parameter refs
	resolveRefs(input)

	// Estimate how long the stackActionToEstimate will take
	// (This is only for spinner output, we calculate total time separately)
	var stackActionToEstimate StackAction
	if input.stackExists {
		stackActionToEstimate = Update
	} else {
		stackActionToEstimate = Create
	}
	est, esterr := GetResourceEstimate(input.typeName, stackActionToEstimate)
	if esterr != nil {
		config.Debugf("could not get estimate: %v", esterr)
		est = 1
	}
	config.Debugf("Got resource estimate for %v: %v", input.logicalId, est)
	spin(input.typeName, input.logicalId, fmt.Sprintf("estimate: %v seconds", est))
	spinner.Pop()

	// Call generic prediction functions that we can run against
	// all resources, even if there is not a predictor.

	spin(input.typeName, input.logicalId, "exists already?")

	code := FG001
	// Make sure the resource does not already exist
	if cfn.ResourceAlreadyExists(input.typeName, input.resource,
		input.stackExists, input.source.Node, input.dc) {
		forecast.Add(code, false, "Already exists")
	} else {
		LineNumber = input.resource.Line
		forecast.Add(code, true, "Does not exist")
	}

	spinner.Pop()

	// Check permissions
	if IncludeIAM {
		err := checkPermissions(input, &forecast)
		if err != nil {
			config.Debugf("Unable to check permissions: %v", err)
			return Forecast{}
		}
	}

	// Check service quotas
	// TODO - Can we do this in a generic way?
	// https://docs.aws.amazon.com/sdk-for-go/api/service/servicequotas/#ServiceQuotas.GetServiceQuota
	// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/servicequotas

	// TODO - What about drift errors? Can we predict what will fail based on
	// a drift detection report for the stack if it already exists?

	// TODO - Regional capabilities. Does this service/feature exist in the region?

	// See if we have a specific forecaster for this type
	fn, found := forecasters[input.typeName]

	if found {
		// Call the prediction function and append the results
		config.Debugf("Running forecaster for %v", input.typeName)
		LineNumber = input.resource.Line
		forecast.Append(fn(input))
	}

	spinner.Pop()

	return forecast
}

// Query the account to make predictions about deployment failures.
// Returns true if no failures are predicted.
func predict(source cft.Template, stackName string, stack types.Stack, stackExists bool, dc *dc.DeployConfig) bool {

	config.Debugf("About to make API calls for failure prediction...")

	spinner.Push("Making predictions")

	// Visit each resource in the template and see if it matches
	// one of our predictions

	forecast := makeForecast("", "")

	rootMap := source.Node.Content[0]

	// Add the --debug arg to see a json version of the yaml node data model for the template
	//config.Debugf("node: %v", toJson(rootMap))

	// Iterate over each resource

	_, resources, _ := s11n.GetMapValue(rootMap, "Resources")
	if resources == nil {
		panic("Expected to find a Resources section in the template")
	}

	for i, r := range resources.Content {

		if i%2 != 0 {
			continue
		}
		logicalId := r.Value
		LineNumber = r.Line
		config.Debugf("logicalId: %v", logicalId)

		resource := resources.Content[i+1]
		_, typeNode, _ := s11n.GetMapValue(resource, "Type")
		if typeNode == nil {
			panic(fmt.Sprintf("Expected %v to have a Type", logicalId))
		}

		// Check the type and call functions that make checks
		// on that type of resource.

		typeName := typeNode.Value // Should be something like AWS::S3::Bucket
		config.Debugf("typeName: %v", typeName)

		spinner.Push(fmt.Sprintf("Checking %s: %s", typeName, logicalId))

		input := PredictionInput{}
		input.logicalId = logicalId
		input.source = source
		input.resource = resource
		input.stackName = stackName
		input.stackExists = stackExists
		input.stack = stack
		input.typeName = typeName
		input.dc = dc
		cfg := aws.Config()
		callerArn, err := iam.GetCallerArn(cfg) // arn:aws:iam::755952356119:role/Admin
		if err != nil {
			panic("unable to get caller arn")
		}
		arnTokens := strings.Split(callerArn, ":")
		if len(arnTokens) != 6 {
			panic(fmt.Sprintf("unexpected number of tokens in caller arn: %v", callerArn))
		}
		input.env = Env{partition: arnTokens[1], region: cfg.Region, account: arnTokens[4]}
		input.roleArn = RoleArn
		if input.roleArn == "" {
			input.roleArn = callerArn
		}

		forecast.Append(forecastForType(input))

		spinner.Pop()
	}

	spinner.Stop()

	// Figure out how long we think the stack will take to execute
	totalSeconds := PredictTotalEstimate(source, stackExists)
	config.Debugf("totalSeconds: %d", totalSeconds)

	if forecast.GetNumFailed() > 0 {
		fmt.Println(console.Red("Stormy weather ahead! 🌪")) // 🌩️⛈
		fmt.Println()
		fmt.Println(console.Red(fmt.Sprintf(
			"%d checks failed out of %d total checks",
			forecast.GetNumFailed(),
			forecast.GetNumChecked())))
		for _, reason := range forecast.Failed {
			fmt.Println(console.Red(reason.String()))
		}
		if all {
			fmt.Println()
			fmt.Println(console.Green(fmt.Sprintf(
				"%d checks passed out of %d total checks",
				forecast.GetNumPassed(),
				forecast.GetNumChecked())))
			for _, reason := range forecast.Passed {
				fmt.Println(console.Green(reason.String()))
			}
		}

		return false
	} else {
		fmt.Println(console.Green(fmt.Sprintf(
			"Clear skies! 🌞 All %d checks passed. Estimated time: %s",
			forecast.GetNumChecked(),
			FormatEstimate(totalSeconds))))
		if all {
			fmt.Println()
			for _, reason := range forecast.Passed {
				fmt.Println(console.Green(reason.String()))
			}
		}
		return true
	}

	// TODO - We might be able to incorporate AWS Config proactive controls here
	// https://aws.amazon.com/blogs/aws/new-aws-config-rules-now-support-proactive-compliance/

	// What about hooks? Could we invoke those handlers to see if they will fail before deployment?

}

// Cmd is the forecast command's entrypoint
var Cmd = &cobra.Command{
	Use:   "forecast --experimental <template> [stackName]",
	Short: "Predict deployment failures",
	Long: `Outputs warnings about potential deployment failures due to constraints in 
the account or misconfigurations in the template related to dependencies in 
the account.

NOTE: This is an experimental feature!

To use this command, add --experimental or -x as an argument.

This command is not a linter! Use cfn-lint for that. The forecast command 
is concerned with things that could go wrong during deployment, after the 
template has been checked to make sure it has a valid syntax.

This command checks for some common issues across all resources, and 
resource-specific checks. See the README for more details.
`,
	Args:                  cobra.RangeArgs(1, 2),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fn := args[0]
		base := filepath.Base(fn)
		var suppliedStackName string

		if len(args) == 2 {
			suppliedStackName = args[1]
		} else {
			suppliedStackName = ""
		}

		// TODO: Remove this when the design stabilizes
		if !Experimental {
			panic("Please add the --experimental arg to use this feature")
		}
		pkg.Experimental = Experimental

		config.Debugf("Generating forecast for %v", fn)

		source, err := pkg.File(fn)
		if err != nil {
			panic(err)
		}

		// Packaging is necessary if we want to forecast a template with
		// modules or anything else that needs packaging.
		// But.. we lost line numbers, so we need to re-parse the file
		content := format.CftToYaml(source)
		source, err = parse.String(content)
		if err != nil {
			panic(err)
		}

		stackName := dc.GetStackName(suppliedStackName, base)

		// Check current stack status
		spinner.Push(fmt.Sprintf("Checking current status of stack '%s'", stackName))
		stack, stackExists := deploy.CheckStack(stackName)
		spinner.Pop()

		msg := ""
		if stackExists {
			msg = "exists"
		} else {
			msg = "does not exist"
		}
		config.Debugf("Stack %v %v", stackName, msg)

		if action == DELETE || action == UPDATE && !stackExists {
			panic(fmt.Sprintf("stack %v does not exist, action %s is not valid", stackName, action))
		}

		if action == CREATE && stackExists {
			panic(fmt.Sprintf("stack %v already exists, action %s is not valid", stackName, action))
		}

		dc, err := dc.GetDeployConfig(tags, params, configFilePath, base,
			source, stack, stackExists, true, false)
		if err != nil {
			panic(err)
		}

		if !predict(source, stackName, stack, stackExists, dc) {
			os.Exit(1)
		}

	},
}

func init() {
	Cmd.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
	Cmd.Flags().BoolVar(&IncludeIAM, "include-iam", false, "Include permissions checks, which can take a long time")
	Cmd.Flags().BoolVarP(&all, "all", "a", false, "Show all checks, not just failed ones")
	Cmd.Flags().BoolVarP(&Experimental, "experimental", "x", false, "Acknowledge that this is an experimental feature")
	Cmd.Flags().StringVar(&RoleArn, "role-arn", "", "An optional execution role arn to use for predicting IAM failures")
	Cmd.Flags().StringVar(&ResourceType, "type", "", "Optional resource type to limit checks to only that type")
	Cmd.Flags().StringSliceVar(&tags, "tags", []string{}, "add tags to the stack; use the format key1=value1,key2=value2")
	Cmd.Flags().StringSliceVar(&params, "params", []string{}, "set parameter values; use the format key1=value1,key2=value2")
	Cmd.Flags().StringVarP(&configFilePath, "config", "c", "", "YAML or JSON file to set tags and parameters")
	Cmd.Flags().StringVar(&action, "action", ALL, "The stack action to check: create, update, delete, all (default is all)")
	Cmd.Flags().StringSliceVar(&ignore, "ignore", []string{}, "Resource types and specific codes to ignore, separated by commas, for example, AWS::S3::Bucket,F0002")

	// If you want to add a prediction for a type that is not already covered, add it here
	// The function must return a Forecast struct
	// For example:
	// forecasters["AWS::New::Type"] = checkTheNewType

	forecasters["AWS::S3::Bucket"] = checkS3Bucket
	forecasters["AWS::S3::BucketPolicy"] = checkS3BucketPolicy
	forecasters["AWS::EC2::Instance"] = checkEC2Instance
	forecasters["AWS::EC2::SecurityGroup"] = checkEC2SecurityGroup
	forecasters["AWS::RDS::DBCluster"] = checkRDSDBCluster
	forecasters["AWS::AutoScaling::LaunchConfiguration"] = checkAutoScalingLaunchConfiguration
	forecasters["AWS::EC2::LaunchTemplate"] = checkEC2LaunchTemplate
	forecasters["AWS::ElasticLoadBalancingV2::Listener"] = checkELBListener
	forecasters["AWS::SNS::Topic"] = checkSNSTopic
	forecasters["AWS::ElasticLoadBalancingV2::TargetGroup"] = checkELBTargetGroup
	forecasters["AWS::Lambda::Function"] = checkLambdaFunction

	// Initialize estimates map
	InitEstimates()

}
