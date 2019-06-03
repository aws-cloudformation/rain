package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/util"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:                   "ls <stack>",
	Short:                 "List running CloudFormation stacks",
	Long:                  "Displays a table of all running stacks or the contents of <stack> if provided.",
	Args:                  cobra.MaximumNArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			table := util.NewTable("Name", "Status")

			stacks, err := cfn.ListStacks()
			if err != nil {
				util.Die(fmt.Errorf("Failed to list stacks: %s", err))
			}

			for _, stack := range stacks {
				table.Append(*stack.StackName, colouriseStatus(string(stack.StackStatus)))
			}

			table.Sort()

			fmt.Println()
			fmt.Println(util.Yellow(fmt.Sprintf("CloudFormation stacks in %s:", client.GetConfig().Region)))
			fmt.Println(table.String())
		} else if len(args) == 1 {
			stackName := args[0]

			stack, err := cfn.GetStack(stackName)
			if err != nil {
				util.Die(fmt.Errorf("Failed to list stack '%s': %s", stackName, err))
			}

			fmt.Println(getStackOutput(stack))
		}
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
