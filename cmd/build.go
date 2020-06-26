package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/aws-cloudformation/rain/cfn/spec"
	"github.com/aws-cloudformation/rain/cfn/spec/builder"
	"github.com/aws-cloudformation/rain/console/text/colourise"
	"github.com/spf13/cobra"
)

var buildListFlag = false

var bareTemplate = false

var buildCmd = &cobra.Command{
	Use:   "build [<resource type>...]",
	Short: "Create CloudFormation templates",
	Long:  "Outputs a CloudFormation template containing the named resource types.",
	Args: func(cmd *cobra.Command, args []string) error {
		if buildListFlag {
			return nil
		}

		return cobra.MinimumNArgs(1)(cmd, args)
	},
	Annotations:           templateAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if buildListFlag {
			types := make([]string, 0)
			for t := range spec.Cfn.ResourceTypes {
				types = append(types, t)
			}
			sort.Strings(types)
			fmt.Println(strings.Join(types, "\n"))

			return
		}

		if len(args) == 0 {
			panic("You didn't specify any resource types to build")
		}

		config := make(map[string]string)
		for _, typeName := range args {
			resourceName := "My" + strings.Split(typeName, "::")[2]
			config[resourceName] = typeName
		}

		b := builder.NewCfnBuilder(!bareTemplate, true)
		t, c := b.Template(config)
		out := format.Anything(t, format.Options{Comments: c})
		out = colourise.Yaml(out)

		fmt.Println(out)
	},
}

func init() {
	buildCmd.Flags().BoolVarP(&buildListFlag, "list", "l", false, "List all CloudFormation resource types")
	buildCmd.Flags().BoolVarP(&bareTemplate, "bare", "b", false, "Produce a minimal template, omitting all optional resource properties")
	Rain.AddCommand(buildCmd)
}
