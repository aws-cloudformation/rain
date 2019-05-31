package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/util"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
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

			err := cfn.ListStacks(func(s cloudformation.StackSummary) {
				table.Append(*s.StackName, colouriseStatus(string(s.StackStatus)))
			})
			if err != nil {
				util.Die(fmt.Errorf("Failed to list stacks: %s", err))
			}

			table.Sort()

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
