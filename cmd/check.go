package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cfn/parse"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:                   "check <template file>",
	Short:                 "Validate a CloudFormation template against the spec",
	Long:                  "Reads the specified CloudFormation template and validates it against the current CloudFormation specification.",
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fn := args[0]

		t, err := parse.File(fn)
		if err != nil {
			panic(fmt.Errorf("Unable to parse template '%s': %s", fn, err))
		}

		out, ok := t.Check()
		if !ok {
			for _, node := range out.Nodes() {
				if node.Content.Comment() != "" {
					fmt.Println(node)
				}
			}
		} else {
			fmt.Println("Template ok")
		}
	},
}

func init() {
	Root.AddCommand(checkCmd)
}
