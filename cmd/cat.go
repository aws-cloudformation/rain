package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/aws-cloudformation/rain/cfn/parse"
	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/console/spinner"
	"github.com/spf13/cobra"
)

var transformed = false
var unformatted = false

var catCmd = &cobra.Command{
	Use:                   "cat <stack>",
	Short:                 "Get the CloudFormation template from a running stack",
	Long:                  "Downloads the template used to deploy <stack> and prints it to stdout.",
	Args:                  cobra.ExactArgs(1),
	Annotations:           stackAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]

		spinner.Status(fmt.Sprintf("Getting template from %s", stackName))
		template, err := cfn.GetStackTemplate(stackName, transformed)
		if err != nil {
			panic(errorf(err, "Failed to get template for stack '%s'", stackName))
		}
		spinner.Stop()

		if unformatted {
			fmt.Println(template)
		} else {
			t, err := parse.String(template)
			if err != nil {
				panic(errorf(err, "Failed to parse template for stack '%s'", stackName))
			}

			fmt.Println(format.Template(t, format.Options{}))
		}
	},
}

func init() {
	catCmd.Flags().BoolVarP(&transformed, "transformed", "t", false, "Get the template with transformations applied by CloudFormation.")
	catCmd.Flags().BoolVarP(&unformatted, "unformatted", "u", false, "Output the template in its raw form and do not attempt to format it.")
	Rain.AddCommand(catCmd)
}
