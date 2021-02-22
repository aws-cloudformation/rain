package rm

import (
	"fmt"
	"os"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/spf13/cobra"
)

var yes bool
var detach bool

// Cmd is the rm command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "rm <stack>",
	Short:                 "Delete a running CloudFormation stack",
	Long:                  "Deletes the CloudFormation stack named <stack> and waits for the action to complete.",
	Args:                  cobra.ExactArgs(1),
	Annotations:           cmd.StackAnnotation,
	Aliases:               []string{"remove", "del", "delete"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]

		spinner.Push("Fetching stack status")
		stack, err := cfn.GetStack(stackName)
		if err != nil {
			panic(ui.Errorf(err, "unable to delete stack '%s'", stackName))
		}

		if *stack.EnableTerminationProtection {
			spinner.Pause()

			if yes || console.Confirm(false, "This stack has termination protection enabled. Do you wish to disable it?") {
				spinner.Push("Disabling termination protection")
				if err := cfn.SetTerminationProtection(stackName, false); err != nil {
					panic(ui.Errorf(err, "unable to set termination protection of stack '%s'", stackName))
				}
				spinner.Pop()
			} else {
				panic(fmt.Errorf("user cancelled deletion of stack '%s'", stackName))
			}

			spinner.Resume()
		}

		if !yes {
			output, _ := ui.GetStackOutput(stack)

			spinner.Pause()
			fmt.Println(output)

			if !console.Confirm(false, "Are you sure you want to delete this stack?") {
				panic(fmt.Errorf("user cancelled deletion of stack '%s'", stackName))
			}
			spinner.Resume()
		}

		spinner.Pop()

		err = cfn.DeleteStack(stackName)
		if err != nil {
			panic(ui.Errorf(err, "unable to delete stack '%s'", stackName))
		}

		if detach {
			fmt.Printf("Detaching. You can check your stack's status with: rain watch %s\n", stackName)
		} else {
			status, messages := ui.WaitForStackToSettle(stackName)
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
	Cmd.Flags().BoolVarP(&detach, "detach", "d", false, "Once removal has started, don't wait around for it to finish.")
	Cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Don't ask questions; just delete")
}
