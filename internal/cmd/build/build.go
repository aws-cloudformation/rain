package build

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aws-cloudformation/rain/cft/build"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/spec"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/spf13/cobra"
)

var buildListFlag = false
var bareTemplate = false
var buildJSON = false
var promptFlag = false

// Cmd is the build command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "build [<resource type> or <prompt>]",
	Short:                 "Create CloudFormation templates",
	Long:                  "Outputs a CloudFormation template containing the named resource types.",
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
			cmd.Help()
			return
		}

		// Invoke Bedrock with Claude 2 to generate the template
		if promptFlag {
			prompt(strings.Join(args, " "))
			return
		}

		resources := resolveResources(args)

		t, err := build.Template(resources, !bareTemplate)
		if err != nil {
			panic(err)
		}

		out := format.String(t, format.Options{
			JSON: buildJSON,
		})

		fmt.Println(out)
	},
}

func init() {
	Cmd.Flags().BoolVarP(&buildListFlag, "list", "l", false, "List all CloudFormation resource types")
	Cmd.Flags().BoolVarP(&promptFlag, "prompt", "p", false, "Generate a template using Bedrock and a prompt")
	Cmd.Flags().BoolVarP(&bareTemplate, "bare", "b", false, "Produce a minimal template, omitting all optional resource properties")
	Cmd.Flags().BoolVarP(&buildJSON, "json", "j", false, "Output the template as JSON (default format: YAML)")
	Cmd.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
}
