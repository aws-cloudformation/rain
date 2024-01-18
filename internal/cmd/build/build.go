package build

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
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

			types, err := cfn.ListResourceTypes()

			if err != nil {
				panic(err)
			}

			for _, t := range types {
				fmt.Println(t)
			}

			return
		}

		if len(args) == 0 {
			cmd.Help()
			return
		}

		// --prompt -p
		// Invoke Bedrock with Claude 2 to generate the template
		if promptFlag {
			prompt(strings.Join(args, " "))
			return
		}

		/*
			resources := resolveResources(args)

			t, err := build.Template(resources, !bareTemplate)
			if err != nil {
				panic(err)
			}

			out := format.String(t, format.Options{
				JSON: buildJSON,
			})

			fmt.Println(out)
		*/
		fmt.Println("TODO")
	},
}

func init() {
	Cmd.Flags().BoolVarP(&buildListFlag, "list", "l", false, "List all CloudFormation resource types")
	Cmd.Flags().BoolVarP(&promptFlag, "prompt", "p", false, "Generate a template using Bedrock and a prompt")
	Cmd.Flags().BoolVarP(&bareTemplate, "bare", "b", false, "Produce a minimal template, omitting all optional resource properties")
	Cmd.Flags().BoolVarP(&buildJSON, "json", "j", false, "Output the template as JSON (default format: YAML)")
	Cmd.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
}
