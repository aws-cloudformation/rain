package rm

import (
	"fmt"
	"os"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/cobra"
)

var yes bool
var detach bool
var roleArn string
var changeset bool

func DeleteChangeSet(stack *types.Stack, changeSetName string) error {
	if !yes {
		spinner.Push("Fetching changeset details")
		cs, err := cfn.GetChangeSet(*stack.StackName, changeSetName)
		if err != nil {
			panic(ui.Errorf(err, "failed to get changeset '%s'", changeSetName))
		}
		spinner.Pop()
		fmt.Printf("Arn: %v\n", *cs.ChangeSetId)
		fmt.Printf("Created: %v\n", cs.CreationTime)
		fmt.Printf("Status: %v/%v\n",
			ui.ColouriseStatus(string(cs.ExecutionStatus)),
			ui.ColouriseStatus(string(cs.Status)))

		fmt.Println()

		// The API allows you to delete a changeset that is still in CREATE_PENDING
		// If you do, the stack becomes stuck and you have to call support to delete it.
		// The AWS console prevents this by graying out buttons.

		if cs.ExecutionStatus == types.ExecutionStatusUnavailable ||
			cs.Status == types.ChangeSetStatusCreatePending ||
			cs.Status == types.ChangeSetStatusCreateInProgress {
			panic(fmt.Errorf("cannot delete changeset '%s' due to current status", *cs.ChangeSetName))
		}

		if !console.Confirm(false, "Are you sure you want to delete this changeset?") {
			panic(fmt.Errorf("user cancelled deletion of changeset '%s'", *cs.ChangeSetName))
		}
	}
	return cfn.DeleteChangeSet(*stack.StackName, changeSetName)
}

// Cmd is the rm command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "rm <stack> [changeset]",
	Short:                 "Delete a CloudFormation stack or changeset",
	Long:                  "Deletes the CloudFormation stack named <stack> and waits for the action to complete. With -c, deletes a changeset named [changeset].",
	Args:                  cobra.MaximumNArgs(2),
	Aliases:               []string{"remove", "del", "delete"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			panic("at least one argument is required")
		}
		stackName := args[0]

		spinner.Push("Fetching stack status")
		stack, err := cfn.GetStack(stackName)
		if err != nil {
			panic(ui.Errorf(err, "unable to get stack '%s'", stackName))
		}
		spinner.Pop()

		if changeset {
			if len(args) != 2 {
				panic("expected 2 arguments: stackName changeSetName")
			}
			if err := DeleteChangeSet(&stack, args[1]); err != nil {
				panic(err)
			}
			return
		}

		if !yes {
			output, _ := cfn.GetStackOutput(stack)

			fmt.Println(output)

			if !console.Confirm(false, "Are you sure you want to delete this stack?") {
				panic(fmt.Errorf("user cancelled deletion of stack '%s'", stackName))
			}
		}

		if *stack.EnableTerminationProtection {

			if yes || console.Confirm(false, "This stack has termination protection enabled. Do you wish to disable it?") {
				spinner.Push("Disabling termination protection")
				if err := cfn.SetTerminationProtection(stackName, false); err != nil {
					panic(ui.Errorf(err, "unable to set termination protection of stack '%s'", stackName))
				}
				spinner.Pop()
			} else {
				panic(fmt.Errorf("user cancelled deletion of stack '%s'", stackName))
			}
		}

		err = cfn.DeleteStack(stackName, roleArn)
		if err != nil {
			panic(ui.Errorf(err, "unable to delete stack '%s'", stackName))
		}

		if detach {
			fmt.Printf("Detaching. You can check your stack's status with: rain watch %s\n", stackName)
		} else {
			status, messages := cfn.WaitForStackToSettle(stackName)
			stack, _ = cfn.GetStack(stackName)

			if status == "DELETE_COMPLETE" {
				fmt.Println(console.Green(fmt.Sprintf("Successfully deleted stack '%s'", stackName)))
				return
			}

			fmt.Fprintln(os.Stderr, console.Red(fmt.Sprintf("Failed to delete stack '%s'", stackName)))

			if len(messages) > 0 {
				fmt.Fprintln(os.Stderr, console.Yellow("Messages:"))
				for _, message := range messages {
					fmt.Fprintf(os.Stderr, "  - %s\n", message)
				}
			}

			os.Exit(1)
		}
	},
}

func init() {
	Cmd.Flags().BoolVarP(&detach, "detach", "d", false, "once removal has started, don't wait around for it to finish")
	Cmd.Flags().BoolVarP(&yes, "yes", "y", false, "don't ask questions; just delete")
	Cmd.Flags().StringVar(&roleArn, "role-arn", "", "ARN of an IAM role that CloudFormation should assume to remove the stack")
	Cmd.Flags().BoolVarP(&changeset, "changeset", "c", false, "delete a changeset")
}
