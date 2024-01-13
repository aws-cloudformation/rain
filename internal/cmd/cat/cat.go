package cat

import (
	"fmt"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/dc"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/spf13/cobra"
)

var transformed = false
var unformatted = false
var config = false

// Cmd is the cat command's entrypoint
var Cmd = &cobra.Command{
	Use:   "cat <stack>",
	Short: "Get the CloudFormation template from a running stack",
	Long: `Downloads the template or the configuration file used to deploy <stack> and prints it to stdout.

The config flag can be used to get the rain config file for the stack instead of the template.
	
Example: 
  // Get the template for the "my-stack" stack 
  rain cat my-stack

  // Get the config file for the "my-stack" stack
  rain cat --config my-stack

`,
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]

		// Output the config file if requested instead of the template
		if config {
			spinner.Push(fmt.Sprintf("Getting config from stack '%s'", stackName))
			stack, err := cfn.GetStack(stackName)
			if err != nil {
				panic(ui.Errorf(err, "failed to get stack '%s'", stackName))
			}
			spinner.Pop()

			deployedConfig, err := dc.ConfigFromStack(stack)
			if err != nil {
				panic(ui.Errorf(err, "unable to get configuration for stack : '%s'", stackName))
			}
			fmt.Print(deployedConfig)

			return
		}

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
	Cmd.Flags().BoolVarP(&transformed, "transformed", "t", false, "get the template with transformations applied by CloudFormation")
	Cmd.Flags().BoolVarP(&unformatted, "unformatted", "u", false, "output the template in its raw form; do not attempt to format it")
	Cmd.Flags().BoolVarP(&config, "config", "c", false, "output the config file for the existing stack")
}
