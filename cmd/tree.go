package cmd

import (
	"fmt"
	"sort"

	"github.com/aws-cloudformation/rain/parse"
	"github.com/aws-cloudformation/rain/template"
	"github.com/aws-cloudformation/rain/util"
	"github.com/spf13/cobra"
)

var allLinks = false

func printLinks(links []interface{}, typeFilter string) {
	names := make([]string, 0)
	for _, link := range links {
		to := link.(template.Element)
		if to.Type == typeFilter {
			names = append(names, to.Name)
		}
	}

	if len(names) == 0 {
		return
	}

	fmt.Printf("    %s:\n", typeFilter)
	for _, name := range names {
		fmt.Printf("      - %s\n", util.Orange(name))
	}
}

func printGraph(graph template.Graph, typeFilter string) {
	froms := make([]template.Element, 0)
	fromLinks := make(map[template.Element][]interface{})

	for _, item := range graph.Items() {
		from := item.(template.Element)
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
		fmt.Printf("  %s:\n", util.Yellow(from.Name))
		printLinks(fromLinks[from], "Parameters")
		printLinks(fromLinks[from], "Resources")
	}

	fmt.Println()
}

var graphCmd = &cobra.Command{
	Use:                   "tree [template]",
	Short:                 "Finds dependencies of resources and outputs in a CloudFormation template",
	Long:                  "Find and display the dependencies between parameters, resources, and outputs in a CloudFormation template.",
	Args:                  cobra.ExactArgs(1),
	Aliases:               []string{"graph"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fileName := args[0]

		input, err := parse.ReadFile(fileName)
		if err != nil {
			panic(fmt.Errorf("Unable to parse template '%s': %s", fileName, err))
		}

		t := template.Template(input)

		graph := t.Graph()
		sort.Sort(graph)

		printGraph(graph, "Resources")
		printGraph(graph, "Outputs")
	},
}

func init() {
	graphCmd.Flags().BoolVarP(&allLinks, "all", "a", false, "Display all elements, even those without any dependencies")
	rootCmd.AddCommand(graphCmd)
}
