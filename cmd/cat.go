package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/spf13/cobra"
)

var catCmd = &cobra.Command{
	Use:                   "cat [stack]",
	Short:                 "Get the CloudFormation template from a deployed stack",
	Long:                  "Downloads the template used to deploy [stack] and prints it to stdout.",
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(cfn.GetStackTemplate(args[0]))
	},
}

func init() {
	rootCmd.AddCommand(catCmd)
}
