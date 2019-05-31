package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/util"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:                   "rm <stack>",
	Short:                 "Delete a running CloudFormation stack",
	Long:                  "Deletes the CloudFormation stack named <stack> and waits for the action to complete.",
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]

		fmt.Printf("Deleting stack '%s'...", stackName)

		if !stackExists(stackName) {
			util.ClearLine()
			util.Die(fmt.Errorf("No such stack '%s'", stackName))
		}

		err := cfn.DeleteStack(stackName)
		if err != nil {
			util.ClearLine()
			util.Die(err)
		}

		status := waitForStackToSettle(stackName)

		if status == "DELETE_COMPLETE" {
			fmt.Println(util.Green("Successfully deleted " + stackName))
		} else {
			fmt.Println(util.Red("Failed to delete " + stackName))
		}

		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
