package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/util"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:                   "rm [stack]",
	Short:                 "Delete a CloudFormation stack",
	Long:                  "Deletes the CloudFormation stack named [stack] and waits for the action to complete.",
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]

		err := cfn.DeleteStack(stackName)
		if err != nil {
			util.Die(err)
		}

		stackId := stackName

		for {
			stack, err := cfn.GetStack(stackId)
			if err != nil {
				util.Die(err)
			}

			// Swap out the stack name for its ID so we can deal with the stack once deleted
			stackId = *stack.StackId

			outputStack(stack, true)

			message := ""

			status := string(stack.StackStatus)

			switch {
			case status == "DELETE_COMPLETE":
				message = "Successfully deleted " + stackName
			case strings.HasSuffix(status, "_COMPLETE") || strings.HasSuffix(status, "_FAILED"):
				message = "Failed to delete " + stackName
			}

			if message != "" {
				fmt.Println()
				fmt.Println(message)
				return
			}

			time.Sleep(2 * time.Second)
		}
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
