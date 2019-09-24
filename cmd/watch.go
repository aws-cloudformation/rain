package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:                   "watch <stack>",
	Short:                 "Display an updating view of a CloudFormation stack",
	Long:                  "Repeatedly displays the status of a CloudFormation stack. Useful for watching the progress of a deployment started from outside of Rain.",
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]

		stack, err := cfn.GetStack(stackName)
		if err != nil {
			panic(fmt.Errorf("Error watching stack '%s': %s", stackName, err))
		}

		if stackHasSettled(stack) {
			fmt.Println(getStackOutput(stack, false))
			fmt.Println("Not watching unchanging stack.")
			return
		}

		fmt.Println("Final stack status:", colouriseStatus(waitForStackToSettle(stackName)))
	},
}

func init() {
	Root.AddCommand(watchCmd)
}
