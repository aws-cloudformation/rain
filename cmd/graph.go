package cmd

import (
	"fmt"
	"sort"

	"github.com/aws-cloudformation/rain/lib"
	"github.com/aws-cloudformation/rain/util"
	"github.com/awslabs/aws-cloudformation-template-formatter/parse"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph [template file]",
	Short: "Graph dependencies between resources in a template",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fileName := args[0]

		input, err := parse.ReadFile(fileName)
		if err != nil {
			util.Die(err)
		}

		template := lib.Template(input)

		graph := make(map[string]map[string]bool)

		for _, dep := range template.FindDependencies() {
			from := dep.From.String()
			to := dep.To.String()

			if _, ok := graph[from]; !ok {
				graph[from] = make(map[string]bool)
			}

			graph[from][to] = true
		}

		froms := make([]string, 0)

		for from, _ := range graph {
			froms = append(froms, from)
		}

		sort.Strings(froms)

		for _, from := range froms {
			fmt.Println(from)

			for to, _ := range graph[from] {
				fmt.Println("  ->", to)
			}

			fmt.Println()
		}
	},
}

func init() {
	rootCmd.AddCommand(graphCmd)
}
