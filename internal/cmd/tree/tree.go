package tree

import (
	"github.com/aws-cloudformation/rain/internal/ui"

	"github.com/aws-cloudformation/rain/cft/graph"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/spf13/cobra"
)

var allLinks = false
var dotGraph = false
var twoWayTree = false

// Cmd is the tree command's entrypoint
var Cmd = &cobra.Command{
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
			panic(ui.Errorf(err, "unable to parse template '%s'", fileName))
		}

		g := graph.New(t)

		if dotGraph {
			printDot(g)
		} else {
			printGraph(g, "Parameters")
			printGraph(g, "Resources")
			printGraph(g, "Outputs")
		}
	},
}

func init() {
	Cmd.Flags().BoolVarP(&allLinks, "all", "a", false, "Display all elements, even those without any dependencies")
	Cmd.Flags().BoolVarP(&twoWayTree, "both", "b", false, "For each element, display both its dependencies and its dependents")
	Cmd.Flags().BoolVarP(&dotGraph, "dot", "d", false, "Output the graph in GraphViz DOT format")
}
