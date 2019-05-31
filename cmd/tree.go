package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/lib"
	"github.com/aws-cloudformation/rain/util"
	"github.com/awslabs/aws-cloudformation-template-formatter/format"
	"github.com/awslabs/aws-cloudformation-template-formatter/parse"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:                   "tree [template]",
	Short:                 "Finds dependencies between entities in a CloudFormation template",
	Long:                  "Find and display the dependencies between parameters, resources, and outputs in a CloudFormation template.",
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fileName := args[0]

		input, err := parse.ReadFile(fileName)
		if err != nil {
			util.Die(err)
		}

		template := lib.Template(input)

		graph := make(map[string]interface{})

		for _, dep := range template.FindDependencies() {
			fromType := fmt.Sprintf("%ss", dep.From.Type)
			toType := fmt.Sprintf("%ss", dep.To.Type)

			fromTypeGraph, ok := graph[fromType].(map[string]interface{})
			if !ok {
				fromTypeGraph = make(map[string]interface{})
				graph[fromType] = fromTypeGraph
			}

			fromNameGraph, ok := fromTypeGraph[dep.From.Name].(map[string]interface{})
			if !ok {
				fromNameGraph = make(map[string]interface{})
				fromTypeGraph[dep.From.Name] = fromNameGraph
			}

			_, ok = fromNameGraph[toType].([]string)
			if !ok {
				fromNameGraph[toType] = make([]string, 0)
			}

			fromNameGraph[toType] = append(fromNameGraph[toType].([]string), dep.To.Name)
		}

		fmt.Println(format.Yaml(graph))
	},
}

func init() {
	// Disabled for now
	// rootCmd.AddCommand(graphCmd)
}
