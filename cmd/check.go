package cmd

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cfn/parse"
	"github.com/aws-cloudformation/rain/console/text"
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
		if ok {
			fmt.Println("Template ok")
		} else {
			fmt.Println("Errors:")

			var path []interface{}

			for _, node := range out.Nodes() {
				if node.Content.Comment() != "" {
					for i, part := range node.Path {
						if i >= len(path) || part != path[i] {
							fmt.Printf("%s%s:",
								strings.Repeat("  ", i+1),
								text.Orange(fmt.Sprint(part)),
							)

							if i == len(node.Path)-1 {
								fmt.Printf(" %s", text.Red(node.Content.Comment()))
							}

							fmt.Println()
						}
					}
					path = node.Path
				}
			}
		}
	},
}

func init() {
	Root.AddCommand(checkCmd)
}
