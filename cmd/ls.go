package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/util"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls [stack name]",
	Short: "List running stacks.",
	Long:  "Displays a table of all running stacks or the contents of a specific, named stack.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			table := util.NewTable("Name", "Status")

			cfn.ListStacks(func(s cloudformation.StackSummary) {
				table.Append(*s.StackName, colouriseStatus(string(s.StackStatus)))
			})

			fmt.Println(table.String())
		} else if len(args) == 1 {
			stack, err := cfn.GetStack(args[0])
			if err != nil {
				util.Die(err)
			}

			outputStack(stack, false)
		}
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
