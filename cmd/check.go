package cmd

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cfn/parse"
	"github.com/aws-cloudformation/rain/cfn/value"
	"github.com/aws-cloudformation/rain/console/text"
	"github.com/spf13/cobra"
)

func formatMessages(m value.Interface) string {
	out := strings.Builder{}

	var path []interface{}

	if m.Comment() != "" {
		out.WriteString(fmt.Sprintf(" %s", text.Red(m.Comment())))
	}

	for _, node := range m.Nodes() {
		if node.Content.Comment() != "" {
			for i, part := range node.Path {
				if i >= len(path) || part != path[i] {
					out.WriteString("\n")

					out.WriteString(fmt.Sprintf("%s%s:",
						strings.Repeat("  ", i+1),
						text.Orange(fmt.Sprint(part)),
					))

					if i == len(node.Path)-1 {
						out.WriteString(fmt.Sprintf(" %s", text.Red(node.Content.Comment())))
					}
				}
			}
			path = node.Path
		}
	}

	return out.String()
}

var checkCmd = &cobra.Command{
	Use:                   "check <template file>",
	Short:                 "Validate a CloudFormation template against the spec",
	Long:                  "Reads the specified CloudFormation template and validates it against the current CloudFormation specification.",
	Args:                  cobra.ExactArgs(1),
	Annotations:           templateAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fn := args[0]

		t, err := parse.File(fn)
		if err != nil {
			panic(fmt.Errorf("Unable to parse template '%s': %s", fn, err))
		}

		out, ok := t.Check()

		messages := formatMessages(out)

		if ok {
			if len(messages) == 0 {
				fmt.Printf("%s: ok\n", fn)
			} else {
				fmt.Printf("%s: ok but with warnings%s", fn, messages)
			}
		} else {
			panic(fmt.Sprintf("%s: not ok%s", fn, messages))
		}
	},
}

func init() {
	Root.AddCommand(checkCmd)
}
