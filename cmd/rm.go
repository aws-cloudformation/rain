package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/console"
	"github.com/aws-cloudformation/rain/console/spinner"
	"github.com/aws-cloudformation/rain/console/text"
	"github.com/spf13/cobra"
)

var forceRm = false

var rmCmd = &cobra.Command{
	Use:                   "rm <stack>",
	Short:                 "Delete a running CloudFormation stack",
	Long:                  "Deletes the CloudFormation stack named <stack> and waits for the action to complete.",
	Args:                  cobra.ExactArgs(1),
	Aliases:               []string{"remove", "del", "delete"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]

		spinner.Status("Checking stack status...")
		stack, err := cfn.GetStack(stackName)
		if err != nil {
			panic(fmt.Errorf("Unable to delete stack '%s': %s", stackName, err))
		}

		if !forceRm {
			output := getStackOutput(stack, false)
			spinner.Stop()

			fmt.Println(output)

			if !console.Confirm(true, "Are you sure you want to delete this stack?") {
				panic(fmt.Errorf("User cancelled deletion of stack '%s'.", stackName))
			}
		}

		spinner.Stop()

		fmt.Printf("Deleting '%s' in %s...\n", stackName, client.Config().Region)

		err = cfn.DeleteStack(stackName)
		if err != nil {
			panic(fmt.Errorf("Unable to delete stack '%s': %s", stackName, err))
		}

		status := waitForStackToSettle(stackName)

		if status == "DELETE_COMPLETE" {
			fmt.Println(text.Green("Successfully deleted " + stackName))
		} else {
			fmt.Println(text.Red("Failed to delete " + stackName))
		}

		fmt.Println()
	},
}

func init() {
	rmCmd.Flags().BoolVarP(&forceRm, "force", "f", false, "Do not ask; just delete")
	rootCmd.AddCommand(rmCmd)
}
