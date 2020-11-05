package watch

import (
	"fmt"
	"time"

	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/ui"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/spf13/cobra"
)

var waitThenWatch = false

// Cmd is the watch command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "watch <stack>",
	Short:                 "Display an updating view of a CloudFormation stack",
	Long:                  "Repeatedly displays the status of a CloudFormation stack. Useful for watching the progress of a deployment started from outside of Rain.",
	Args:                  cobra.ExactArgs(1),
	Annotations:           cmd.StackAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]

		first := true
		for {
			stack, err := cfn.GetStack(stackName)
			if err != nil {
				panic(ui.Errorf(err, "error watching stack '%s'", stackName))
			}

			if !ui.StackHasSettled(stack) {
				// Stack is changing
				break
			}

			if !waitThenWatch {
				// Not changing, not waiting for it
				status, _ := ui.GetStackOutput(stack)
				fmt.Println(status)
				fmt.Println("Not watching unchanging stack.")
				return
			}

			if first {
				spinner.Push("Waiting for stack to begin changing")
				first = false
			}

			time.Sleep(time.Second * 2)
		}

		spinner.Pop()

		status, messages := ui.WaitForStackToSettle(stackName)

		fmt.Println("Final stack status:", ui.ColouriseStatus(status))

		if len(messages) > 0 {
			fmt.Println()
			fmt.Println(console.Yellow("Messages:"))
			for _, message := range messages {
				fmt.Printf("  - %s\n", message)
			}
		}
	},
}

func init() {
	Cmd.Flags().BoolVarP(&waitThenWatch, "wait", "w", false, "Wait for changes to begin rather than refusing to watch an unchanging stack")
}
