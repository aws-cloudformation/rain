package cmd

import (
	"fmt"
	"time"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/console/spinner"
	"github.com/spf13/cobra"
)

var waitThenWatch = false

var watchCmd = &cobra.Command{
	Use:                   "watch <stack>",
	Short:                 "Display an updating view of a CloudFormation stack",
	Long:                  "Repeatedly displays the status of a CloudFormation stack. Useful for watching the progress of a deployment started from outside of Rain.",
	Args:                  cobra.ExactArgs(1),
	Annotations:           stackAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]

		stack, err := cfn.GetStack(stackName)
		if err != nil {
			panic(fmt.Errorf("Error watching stack '%s': %s", stackName, err))
		}

		if stackHasSettled(stack) {
			if waitThenWatch {
				spinner.Status("Waiting for stack to begin changing")
				for {
					time.Sleep(time.Second * 2)
					if !stackHasSettled(stack) {
						break
					}
				}
				spinner.Stop()
			} else {
				fmt.Println(getStackOutput(stack, false))
				fmt.Println("Not watching unchanging stack.")
				return
			}
		}

		fmt.Println("Final stack status:", colouriseStatus(waitForStackToSettle(stackName)))
	},
}

func init() {
	watchCmd.Flags().BoolVarP(&waitThenWatch, "wait", "w", false, "Wait for changes to begin rather than refusing to watch an unchanging stack")
	Rain.AddCommand(watchCmd)
}
