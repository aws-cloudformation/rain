package deploy

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	cftpkg "github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/dc"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
)

var detach bool
var yes bool
var params []string
var tags []string
var configFilePath string
var terminationProtection bool
var keep bool
var roleArn string
var ignoreUnknownParams bool
var noexec bool
var changeset bool
var experimental bool
var includeNested bool

// Cmd is the deploy command's entrypoint
var Cmd = &cobra.Command{
	Use:   "deploy <template> [stack]",
	Short: "Deploy a CloudFormation stack or changeset from a local template",
	Long: `Creates or updates a CloudFormation stack named <stack> from the template file <template>. 
You can also create and execute changesets with this command.
If you don't specify a stack name, rain will use the template filename minus its extension.

If a template needs to be packaged before it can be deployed, rain will package the template first.
Rain will attempt to create an S3 bucket to store artifacts that it packages and deploys.
The bucket's name will be of the format rain-artifacts-<AWS account id>-<AWS region>.

You can configure parameters and tags for your stack in several ways. The command-line flags --params
and --tags always take precedence over other sources. If you specify a config file using the --config
flag, parameters and tags in this file will be used unless they are overridden by command-line
arguments.

The config file format is similar to the AWS CodePipeline "Template configuration file", but it
does not include the 'StackPolicy' key, and may be in either YAML or JSON format.

If a parameter or tag is not specified through command-line flags or the config file,
you can also provide defaults through environment variables. Use variables prefixed with
RAIN_VAR_* for parameters, and RAIN_DEFAULT_TAG_* for tags.

For reference, here are example formats for the config file:

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

To create a changeset (with optional stackName and changeSetName):

rain deploy --no-exec <template> [stackName] [changeSetName]

To execute a changeset:

rain deploy --changeset <stackName> <changeSetName>

To list and delete changesets, use the ls and rm commands.
`,
	Args:                  cobra.RangeArgs(1, 3),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {

		var stackName, changeSetName, fn string
		var err error
		var stack types.Stack
		var templateNode *yaml.Node

		if changeset {

			if len(args) != 2 {
				panic("expected 2 args: rain deploy --changeset <stackName> <changeSetName>")
			}

			stackName = args[0]
			changeSetName = args[1]

		} else {

			fn = args[0]
			base := filepath.Base(fn)

			var suppliedStackName string

			if len(args) >= 2 {
				suppliedStackName = args[1]
			}

			// Optionally name the change set
			if len(args) == 3 {
				changeSetName = args[2]
			}

			// Package template
			if experimental {
				cftpkg.Experimental = true
			}
			spinner.Push(fmt.Sprintf("Preparing template '%s'", base))
			template := PackageTemplate(fn, yes)
			templateNode = template.Node
			spinner.Pop()

			// Before deploying, check to see if there are any Metadata sections.
			// If so, stop if the --experimental flag is not set
			if HasRainMetadata(template) && !experimental {
				panic("metadata commands require the --experimental flag")
			}

			// Process metadata Rain Content before (Run build scripts before deployment)
			if !changeset {
				err := processMetadataBefore(cft.Template{Node: templateNode},
					stackName, filepath.Dir(fn))
				if err != nil {
					panic(err)
				}
			}

			stackName = dc.GetStackName(suppliedStackName, base)

			// Check current stack status
			spinner.Push(fmt.Sprintf("Checking current status of stack '%s'", stackName))
			stack, stackExists := CheckStack(stackName)
			spinner.Pop()

			dc, err := dc.GetDeployConfig(tags, params, configFilePath, base,
				template, stack, stackExists, yes, ignoreUnknownParams)
			if err != nil {
				panic(err)
			}

			// Figure out how long we think the stack will take to execute
			//totalSeconds := forecast.PredictTotalEstimate(template, stackExists)
			// TODO - Wait until the forecast command is GA and add this to output

			// Create change set
			spinner.Push("Creating change set")
			var createErr error
			ctx := cfn.ChangeSetContext{
				Template:      template,
				Params:        dc.Params,
				Tags:          dc.Tags,
				StackName:     stackName,
				ChangeSetName: changeSetName,
				RoleArn:       roleArn,
				IncludeNested: includeNested,
			}
			config.Debugf("ChangeSetContext: %+v", ctx)
			changeSetName, createErr = cfn.CreateChangeSet(&ctx)
			if createErr != nil {
				if changeSetHasNoChanges(createErr.Error()) {
					spinner.Pop()
					fmt.Println(console.Green("Change set was created, but there is no change. Deploy was skipped."))
					return
				} else {
					panic(ui.Errorf(createErr, "error creating changeset"))
				}
			}
			spinner.Pop()

			// Display changeset and exit
			if noexec {
				spinner.Push("Formatting change set")
				status := formatChangeSet(stackName, changeSetName)
				spinner.Pop()

				fmt.Println("Changeset contains the following changes:")
				fmt.Println(status)

				fmt.Println("changeset created but not executed:", changeSetName)
				return
			}

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
						err = cfn.DeleteStack(stackName, "")
						if err != nil {
							panic(ui.Errorf(err, "error deleting empty stack '%s'", stackName))
						}
					}

					panic(errors.New("user cancelled deployment"))
				}
			}

		}

		// Deploy!
		err = cfn.ExecuteChangeSet(stackName, changeSetName, keep)
		if err != nil {
			panic(ui.Errorf(err, "error while executing changeset '%s'", changeSetName))
		}

		if detach {
			fmt.Printf("Detaching. You can check your stack's status with: rain watch %s\n", stackName)
		} else {
			if changeset {
				fmt.Printf("Executing changeset '%s' as stack '%s' in %s.\n",
					changeSetName, stackName, aws.Config().Region)
			} else {
				fmt.Printf("Deploying template '%s' as stack '%s' in %s.\n",
					filepath.Base(fn), stackName, aws.Config().Region)
			}
			status, messages := cfn.WaitForStackToSettle(stackName)
			stack, _ = cfn.GetStack(stackName)
			output := cfn.GetStackSummary(stack, false)

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
			} else if status == "IMPORT_COMPLETE" {
				fmt.Println(console.Green("Successfully imported " + stackName))
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

		// Process Rain Metadata commands (Content)
		if !changeset {
			err := processMetadataAfter(cft.Template{Node: templateNode},
				stackName, filepath.Dir(fn))
			if err != nil {
				panic(err)
			}
		}
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		params = nil
	},
}

func changeSetHasNoChanges(msg string) bool {
	// mesages returned as error when the change set is empty
	noChangeFoundMsg := []string{
		"The submitted information didn't contain changes. Submit different information to create a change set.",
		"No updates are to be performed.",
	}
	for _, m := range noChangeFoundMsg {
		if m == msg {
			return true
		}
	}
	return false
}

// hasRainMetadata returns true if the template has a resource
// with a Metadata section with a Rain node
func HasRainMetadata(template *cft.Template) bool {
	if template.Node.Content[0].Kind == yaml.DocumentNode {
		template.Node = template.Node.Content[0]
	}
	resources, err := template.GetSection(cft.Resources)
	if err != nil {
		config.Debugf("unexpected error getting resource section: %v", err)
		return false
	}
	for i := 0; i < len(resources.Content); i += 2 {
		resource := resources.Content[i+1]
		_, n, _ := s11n.GetMapValue(resource, "Metadata")
		if n == nil {
			continue
		}
		_, n, _ = s11n.GetMapValue(n, "Rain")
		if n == nil {
			continue
		}
		return true
	}
	return false
}

func init() {

	Cmd.Flags().BoolVarP(&detach, "detach", "d", false, "once deployment has started, don't wait around for it to finish")
	Cmd.Flags().BoolVarP(&yes, "yes", "y", false, "don't ask questions; just deploy")
	Cmd.Flags().StringSliceVar(&tags, "tags", []string{}, "add tags to the stack; use the format key1=value1,key2=value2")
	Cmd.Flags().StringSliceVar(&params, "params", []string{}, "set parameter values; use the format key1=value1,key2=value2")
	Cmd.Flags().StringVarP(&configFilePath, "config", "c", "", "YAML or JSON file to set tags and parameters")
	Cmd.Flags().BoolVarP(&terminationProtection, "termination-protection", "t", false, "enable termination protection on the stack")
	Cmd.Flags().BoolVarP(&keep, "keep", "k", false, "keep deployed resources after a failure by disabling rollbacks")
	Cmd.Flags().StringVarP(&roleArn, "role-arn", "", "", "ARN of an IAM role that CloudFormation should assume to deploy the stack")
	Cmd.Flags().BoolVarP(&ignoreUnknownParams, "ignore-unknown-params", "", false, "Ignore unknown parameters")
	Cmd.Flags().BoolVarP(&noexec, "no-exec", "x", false, "do not execute the changeset")
	Cmd.Flags().BoolVar(&changeset, "changeset", false, "execute the changeset, rain deploy --changeset <stackName> <changeSetName>")
	Cmd.Flags().StringVar(&format.NodeStyle, "node-style", "original", format.NodeStyleDocs)
	Cmd.Flags().BoolVar(&experimental, "experimental", false, "Acknowledge that you want to deploy with an experimental feature")
	Cmd.Flags().BoolVar(&includeNested, "nested-change-set", true, "Whether or not to include nested stacks in the change set")
	Cmd.Flags().BoolVar(&cftpkg.NoAnalytics, "no-analytics", false, "Do not write analytics to Metadata")
}
