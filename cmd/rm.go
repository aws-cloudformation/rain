package cmd

import (
	"fmt"

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

		status := waitForStackToSettle(stackName)

		fmt.Println()

		if status == "DELETE_COMPLETE" {
			fmt.Println("Successfully deleted " + stackName)
		} else {
			fmt.Println("Failed to delete " + stackName)
		}

		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
