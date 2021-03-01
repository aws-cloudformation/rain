package deploy

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"

	"github.com/awslabs/smithy-go/ptr"
	"github.com/spf13/cobra"
)

var detach bool
var yes bool
var params []string
var tags []string

// Cmd is the deploy command's entrypoint
var Cmd = &cobra.Command{
	Use:   "deploy <template> [stack]",
	Short: "Deploy a CloudFormation stack from a local template",
	Long: `Creates or updates a CloudFormation stack named <stack> from the template file <template>.
If you don't specify a stack name, rain will use the template filename minus its extension.

If a template needs to be packaged before it can be deployed, rain will package the template first.
Rain will attempt to create an S3 bucket to store artifacts that it packages and deploys.
The bucket's name will be of the format rain-artifacts-<AWS account id>-<AWS region>`,
	Args:                  cobra.RangeArgs(1, 2),
	Annotations:           cmd.StackAnnotation,
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
		parsedTags := listToMap("tag", tags)

		// Parse params
		parsedParams := listToMap("param", params)

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
		parameters := getParameters(template, parsedParams, stack.Parameters, stackExists)

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
		changeSetName, createErr := cfn.CreateChangeSet(template, parameters, parsedTags, stackName)
		if createErr != nil {
			panic(ui.Errorf(createErr, "error creating changeset"))
		}

		changeSetStatus, err := cfn.GetChangeSet(stackName, changeSetName)
		if err != nil {
			panic(ui.Errorf(err, "error getting changeset status '%s'", formatChangeSet(changeSetStatus)))
		}

		spinner.Pop()

		// Confirm changes
		if !yes {
			fmt.Println("CloudFormation will make the following changes:")
			fmt.Println(formatChangeSet(changeSetStatus))

			if !console.Confirm(true, "Do you wish to continue?") {
				err = cfn.DeleteChangeSet(stackName, changeSetName)
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
		err = cfn.ExecuteChangeSet(stackName, changeSetName)
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
	},
}

func init() {
	fixStackNameRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

	Cmd.Flags().BoolVarP(&detach, "detach", "d", false, "Once deployment has started, don't wait around for it to finish.")
	Cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Don't ask questions; just deploy.")
	Cmd.Flags().StringSliceVar(&tags, "tags", []string{}, "Add tags to the stack. Use the format key1=value1,key2=value2.")
	Cmd.Flags().StringSliceVar(&params, "params", []string{}, "Set parameter values. Use the format key1=value1,key2=value2.")
}
