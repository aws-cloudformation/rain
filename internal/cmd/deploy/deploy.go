package deploy

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/dc"
	"github.com/aws-cloudformation/rain/internal/ui"

	"github.com/spf13/cobra"
)

const noChangeFoundMsg = "The submitted information didn't contain changes. Submit different information to create a change set."

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

		var suppliedStackName string

		if len(args) == 2 {
			suppliedStackName = args[1]
		} else {
			suppliedStackName = ""
		}

		// Package template
		spinner.Push(fmt.Sprintf("Preparing template '%s'", base))
		template := PackageTemplate(fn, yes)
		spinner.Pop()

		stackName := dc.GetStackName(suppliedStackName, base)

		// Check current stack status
		spinner.Push(fmt.Sprintf("Checking current status of stack '%s'", stackName))
		stack, stackExists := CheckStack(stackName)
		spinner.Pop()

		dc, err := dc.GetDeployConfig(tags, params, configFilePath, base,
			template, stack, stackExists, yes)
		if err != nil {
			panic(err)
		}

		// Create change set
		spinner.Push("Creating change set")
		changeSetName, createErr := cfn.CreateChangeSet(template, dc.Params, dc.Tags, stackName, roleArn)
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
					err = cfn.DeleteStack(stackName, "")
					if err != nil {
						panic(ui.Errorf(err, "error deleting empty stack '%s'", stackName))
					}
				}

				panic(errors.New("user cancelled deployment"))
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
			fmt.Printf("Deploying template '%s' as stack '%s' in %s.\n", filepath.Base(fn), stackName, aws.Config().Region)

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

	Cmd.Flags().BoolVarP(&detach, "detach", "d", false, "once deployment has started, don't wait around for it to finish")
	Cmd.Flags().BoolVarP(&yes, "yes", "y", false, "don't ask questions; just deploy")
	Cmd.Flags().StringSliceVar(&tags, "tags", []string{}, "add tags to the stack; use the format key1=value1,key2=value2")
	Cmd.Flags().StringSliceVar(&params, "params", []string{}, "set parameter values; use the format key1=value1,key2=value2")
	Cmd.Flags().StringVarP(&configFilePath, "config", "c", "", "YAML or JSON file to set tags and parameters")
	Cmd.Flags().BoolVarP(&terminationProtection, "termination-protection", "t", false, "enable termination protection on the stack")
	Cmd.Flags().BoolVarP(&keep, "keep", "k", false, "keep deployed resources after a failure by disabling rollbacks")
	Cmd.Flags().StringVarP(&roleArn, "role-arn", "", "", "ARN of an IAM role that CloudFormation should assume to deploy the stack")
}
