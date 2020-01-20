package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cfn"
	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/aws-cloudformation/rain/cfn/spec/builder"
	"github.com/aws-cloudformation/rain/console/text/colourise"
	"github.com/spf13/cobra"
)

var docCmd = &cobra.Command{
	Use:                   "doc <resource type>",
	Short:                 "Get documentation for a resource type",
	Long:                  "Displays documentation for the resource type you specify.",
	Args:                  cobra.ExactArgs(1),
	Aliases:               []string{"docs"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		typeName := args[0]

		b := builder.NewCfnBuilder(true, true)
		template, _ := b.Template(map[string]string{
			"MyResource": typeName,
		})

		t := cfn.Template(template)

		out := format.Template(t, format.Options{})

		out = colourise.Yaml(out)

		fmt.Println(out)
	},
}

func init() {
	Root.AddCommand(docCmd)
}
