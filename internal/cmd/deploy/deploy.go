package deploy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"gopkg.in/yaml.v3"

	"github.com/aws/smithy-go/ptr"
	"github.com/spf13/cobra"
)

const noChangeFoundMsg = "The submitted information didn't contain changes. Submit different information to create a change set."

type configFileFormat struct {
	Parameters map[string]string `yaml:"Parameters"`
	Tags       map[string]string `yaml:"Tags"`
}

var detach bool
var yes bool
var params []string
var tags []string
var configFilePath string
var terminationProtection bool
var keep bool
var roleArn string

// Cmd is the deploy command's entrypoint
var Cmd = &cobra.Command{
	Use:   "deploy <template> [stack]",
	Short: "Deploy a CloudFormation stack from a local template",
	Long: `Creates or updates a CloudFormation stack named <stack> from the template file <template>.
If you don't specify a stack name, rain will use the template filename minus its extension.

If a template needs to be packaged before it can be deployed, rain will package the template first.
Rain will attempt to create an S3 bucket to store artifacts that it packages and deploys.
The bucket's name will be of the format rain-artifacts-<AWS account id>-<AWS region>.

The config flag can be used to programmatically set tags and parameters.
The format is similar to the "Template configuration file" for AWS CodePipeline just without the
'StackPolicy' key. The file can be in YAML or JSON format.

JSON:
  {
    "Parameters" : {
      "NameOfTemplateParameter" : "ValueOfParameter",
      ...
    },
    "Tags" : {
      "TagKey" : "TagValue",
      ...
    }
  }

YAML:
  Parameters:
    NameOfTemplateParameter: ValueOfParameter
    ...
  Tags:
    TagKey: TagValue
    ...
`,
	Args:                  cobra.RangeArgs(1, 2),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fn := args[0]
		base := filepath.Base(fn)

		var stackName string

		if len(args) == 2 {
			stackName = args[1]
		} else {
			stackName = base[:len(base)-len(filepath.Ext(base))]

			// Now ensure it's a valid cfc name
			stackName = fixStackNameRe.ReplaceAllString(stackName, "-")

			if len(stackName) > maxStackNameLength {
				stackName = stackName[:maxStackNameLength]
			}
		}

		// Parse tags
		parsedTagFlag := ListToMap("tag", tags)

		// Parse params
		parsedParamFlag := ListToMap("param", params)

		var combinedTags map[string]string
		var combinedParameters map[string]string

		if len(configFilePath) != 0 {
			configFileContent, err := os.ReadFile(configFilePath)
			if err != nil {
				panic(ui.Errorf(err, "unable to read config file '%s'", configFilePath))
			}

			var configFile configFileFormat
			err = yaml.Unmarshal([]byte(configFileContent), &configFile)
			if err != nil {
				panic(ui.Errorf(err, "unable to parse yaml in '%s'", configFilePath))
			}

			combinedTags = configFile.Tags
			combinedParameters = configFile.Parameters

			for k, v := range parsedTagFlag {
				if _, ok := combinedTags[k]; ok {
					fmt.Println(console.Yellow(fmt.Sprintf("tags flag overrides tag in config file: %s", k)))
				}
				combinedTags[k] = v
			}

			for k, v := range parsedParamFlag {
				if _, ok := combinedParameters[k]; ok {
					fmt.Println(console.Yellow(fmt.Sprintf("params flag overrides parameter in config file: %s", k)))
				}
				combinedParameters[k] = v
			}
		} else {
			combinedTags = parsedTagFlag
			combinedParameters = parsedParamFlag
		}

		// Package template
		spinner.Push(fmt.Sprintf("Preparing template '%s'", base))
		template := packageTemplate(fn, yes)
		spinner.Pop()

		// Check current stack status
		spinner.Push(fmt.Sprintf("Checking current status of stack '%s'", stackName))
		stack, stackExists := checkStack(stackName)
		spinner.Pop()

		// Parse params
		config.Debugf("Handling parameters")
		parameters := getParameters(template, combinedParameters, stack.Parameters, stackExists)

		if config.Debug {
			for _, param := range parameters {
				val := ptr.ToString(param.ParameterValue)
				if ptr.ToBool(param.UsePreviousValue) {
					val = "<previous value>"
				}
				config.Debugf("  %s: %s", ptr.ToString(param.ParameterKey), val)
			}
		}

		// Create change set
		spinner.Push("Creating change set")
		changeSetName, createErr := cfn.CreateChangeSet(template, parameters, combinedTags, stackName, roleArn)
		if createErr != nil {
			if createErr.Error() == noChangeFoundMsg {
				spinner.Pop()
				fmt.Println(console.Green("Change set was created, but there is no change. Deploy was skipped."))
				return
			} else {
				panic(ui.Errorf(createErr, "error creating changeset"))
			}
		}
		spinner.Pop()

		// Confirm changes
		if !yes {
			spinner.Push("Formatting change set")
			status := formatChangeSet(stackName, changeSetName)
			spinner.Pop()

			fmt.Println("CloudFormation will make the following changes:")
			fmt.Println(status)

			if !console.Confirm(true, "Do you wish to continue?") {
				err := cfn.DeleteChangeSet(stackName, changeSetName)
				if err != nil {
					panic(ui.Errorf(err, "error while deleting changeset '%s'", changeSetName))
				}

				if !stackExists {
					err = cfn.DeleteStack(stackName)
					if err != nil {
						panic(ui.Errorf(err, "error deleting empty stack '%s'", stackName))
					}
				}

				panic(errors.New("user cancelled deployment"))
			}
		}

		// Deploy!
		err := cfn.ExecuteChangeSet(stackName, changeSetName, keep)
		if err != nil {
			panic(ui.Errorf(err, "error while executing changeset '%s'", changeSetName))
		}

		if detach {
			fmt.Printf("Detaching. You can check your stack's status with: rain watch %s\n", stackName)
		} else {
			fmt.Printf("Deploying template '%s' as stack '%s' in %s.\n", filepath.Base(fn), stackName, aws.Config().Region)

			status, messages := ui.WaitForStackToSettle(stackName)
			stack, _ = cfn.GetStack(stackName)
			output := ui.GetStackSummary(stack, false)

			fmt.Println(output)

			if len(messages) > 0 {
				fmt.Println(console.Yellow("Messages:"))
				for _, message := range messages {
					fmt.Printf("  - %s\n", message)
				}
			}

			if status == "CREATE_COMPLETE" {
				fmt.Println(console.Green("Successfully deployed " + stackName))
			} else if status == "UPDATE_COMPLETE" {
				fmt.Println(console.Green("Successfully updated " + stackName))
			} else {
				panic(fmt.Errorf("failed deploying stack '%s'", stackName))
			}
		}

		// Enable termination protection
		if terminationProtection {
			err = cfn.SetTerminationProtection(stackName, true)
			if err != nil {
				panic(ui.Errorf(err, "error while enabling termination protection on stack '%s'", stackName))
			}
		}
	},
}

func init() {
	fixStackNameRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

	Cmd.Flags().BoolVarP(&detach, "detach", "d", false, "once deployment has started, don't wait around for it to finish")
	Cmd.Flags().BoolVarP(&yes, "yes", "y", false, "don't ask questions; just deploy")
	Cmd.Flags().StringSliceVar(&tags, "tags", []string{}, "add tags to the stack; use the format key1=value1,key2=value2")
	Cmd.Flags().StringSliceVar(&params, "params", []string{}, "set parameter values; use the format key1=value1,key2=value2")
	Cmd.Flags().StringVarP(&configFilePath, "config", "c", "", "YAML or JSON file to set tags and parameters")
	Cmd.Flags().BoolVarP(&terminationProtection, "termination-protection", "t", false, "enable termination protection on the stack")
	Cmd.Flags().BoolVarP(&keep, "keep", "k", false, "keep deployed resources after a failure by disabling rollbacks")
	Cmd.Flags().StringVarP(&roleArn, "role-arn", "", "", "ARN of an IAM role that CloudFormation should assume to deploy the stack")
}
