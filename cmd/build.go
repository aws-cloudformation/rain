package cmd

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/aws-cloudformation/rain/cfn/spec/builder"
	"github.com/aws-cloudformation/rain/console/text/colourise"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:                   "build [<resource type>...]",
	Short:                 "Create CloudFormation templates",
	Long:                  "Outputs a CloudFormation template containing the named resource types.",
	Args:                  cobra.MinimumNArgs(1),
	Aliases:               []string{"docs"},
	Annotations:           templateAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		config := make(map[string]string)
		for _, typeName := range args {
			resourceName := "My" + strings.Split(typeName, "::")[2]
			config[resourceName] = typeName
		}

		b := builder.NewCfnBuilder(true, true)
		t, c := b.Template(config)
		out := format.Anything(t, format.Options{Comments: c})
		out = colourise.Yaml(out)

		fmt.Println(out)
	},
}

func init() {
	Root.AddCommand(buildCmd)
}
