package cat

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/ui"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/spf13/cobra"
)

var transformed = false
var unformatted = false

// Cmd is the cat command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "cat <stack>",
	Short:                 "Get the CloudFormation template from a running stack",
	Long:                  "Downloads the template used to deploy <stack> and prints it to stdout.",
	Args:                  cobra.ExactArgs(1),
	Annotations:           cmd.StackAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]

		spinner.Push(fmt.Sprintf("Getting template from stack '%s'", stackName))
		template, err := cfn.GetStackTemplate(stackName, transformed)
		if err != nil {
			panic(ui.Errorf(err, "failed to get template for stack '%s'", stackName))
		}
		spinner.Pop()

		if unformatted {
			fmt.Println(template)
		} else {
			t, err := parse.String(template)
			if err != nil {
				panic(ui.Errorf(err, "failed to parse template for stack '%s'", stackName))
			}

			fmt.Print(format.String(t, format.Options{}))
		}
	},
}

func init() {
	Cmd.Flags().BoolVarP(&transformed, "transformed", "t", false, "Get the template with transformations applied by CloudFormation.")
	Cmd.Flags().BoolVarP(&unformatted, "unformatted", "u", false, "Output the template in its raw form and do not attempt to format it.")
}
