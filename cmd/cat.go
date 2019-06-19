package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/util"
	"github.com/spf13/cobra"
)

var catCmd = &cobra.Command{
	Use:                   "cat <stack>",
	Short:                 "Get the CloudFormation template from a running stack",
	Long:                  "Downloads the template used to deploy <stack> and prints it to stdout.",
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]

		util.SpinStatus(fmt.Sprintf("Getting template from %s...", stackName))
		template, err := cfn.GetStackTemplate(stackName)
		if err != nil {
			panic(fmt.Errorf("Failed to get template for stack '%s': %s", stackName, err))
		}
		util.SpinStop()

		fmt.Println(template)
	},
}

func init() {
	rootCmd.AddCommand(catCmd)
}
