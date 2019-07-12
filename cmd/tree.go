package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cfn"
	"github.com/aws-cloudformation/rain/cfn/graph"
	"github.com/aws-cloudformation/rain/cfn/parse"
	"github.com/aws-cloudformation/rain/console/text"
	"github.com/spf13/cobra"
)

var allLinks = false

func printLinks(links []interface{}, typeFilter string) {
	names := make([]string, 0)
	for _, link := range links {
		to := link.(cfn.Element)
		if to.Type == typeFilter {
			names = append(names, to.Name)
		}
	}

	if len(names) == 0 {
		return
	}

	fmt.Printf("    %s:\n", typeFilter)
	for _, name := range names {
		fmt.Printf("      - %s\n", text.Orange(name))
	}
}

func printGraph(graph graph.Graph, typeFilter string) {
	froms := make([]cfn.Element, 0)
	fromLinks := make(map[cfn.Element][]interface{})

	for _, item := range graph.Nodes() {
		from := item.(cfn.Element)
		if from.Type == typeFilter {
			links := graph.Get(item)

			if !allLinks && len(links) == 0 {
				continue
			}

			froms = append(froms, from)
			fromLinks[from] = links
		}
	}

	if len(froms) == 0 {
		return
	}

	fmt.Printf("%s:\n", typeFilter)

	for _, from := range froms {
		fmt.Printf("  %s:\n", text.Yellow(from.Name))
		printLinks(fromLinks[from], "Parameters")
		printLinks(fromLinks[from], "Resources")
	}

	fmt.Println()
}

var graphCmd = &cobra.Command{
	Use:                   "tree [template]",
	Short:                 "Find dependencies of Resources and Outputs in a local template",
	Long:                  "Find and display the dependencies between Parameters, Resources, and Outputs in a CloudFormation template.",
	Args:                  cobra.ExactArgs(1),
	Aliases:               []string{"graph"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fileName := args[0]

		t, err := parse.File(fileName)
		if err != nil {
			panic(fmt.Errorf("Unable to parse template '%s': %s", fileName, err))
		}

		graph := t.Graph()

		printGraph(graph, "Resources")
		printGraph(graph, "Outputs")
	},
}

func init() {
	graphCmd.Flags().BoolVarP(&allLinks, "all", "a", false, "Display all elements, even those without any dependencies")
	rootCmd.AddCommand(graphCmd)
}
