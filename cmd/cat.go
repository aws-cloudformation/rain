package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/spf13/cobra"
)

var catCmd = &cobra.Command{
	Use:   "cat [stack name]",
	Short: "Get templates from stacks",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(cfn.GetStackTemplate(args[0]))
	},
}

func init() {
	rootCmd.AddCommand(catCmd)
}
