package tree

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cft/graph"
	"github.com/aws-cloudformation/rain/internal/console"
)

func printLinks(links []graph.Node, typeFilter string) {
	names := make([]string, 0)
	for _, to := range links {
		if to.Type == typeFilter {
			names = append(names, to.Name)
		}
	}

	if len(names) == 0 {
		return
	}

	fmt.Printf("      %s:\n", typeFilter)
	for _, name := range names {
		fmt.Printf("        - %s\n", console.Blue(name))
	}
}

func printGraph(g graph.Graph, typeFilter string) {
	elements := make([]graph.Node, 0)
	fromLinks := make(map[graph.Node][]graph.Node)
	toLinks := make(map[graph.Node][]graph.Node)

	for _, el := range g.Nodes() {
		if el.Type == typeFilter {
			elements = append(elements, el)
			froms := g.Get(el)

			if allLinks || len(froms) > 0 {
				fromLinks[el] = froms
			}

			if twoWayTree {
				tos := g.GetReverse(el)

				if allLinks || len(tos) > 0 {
					toLinks[el] = tos
				}
			}
		}
	}

	if len(fromLinks) == 0 && len(toLinks) == 0 {
		return
	}

	fmt.Printf("%s:\n", typeFilter)

	for _, el := range elements {
		if !allLinks && len(fromLinks[el]) == 0 && len(toLinks[el]) == 0 {
			continue
		}

		fmt.Printf("  %s:\n", console.Yellow(el.Name))

		if allLinks || len(fromLinks[el]) > 0 {
			if len(fromLinks[el]) == 0 {
				fmt.Println("    DependsOn: []")
			} else {
				fmt.Println("    DependsOn:")
				printLinks(fromLinks[el], "Parameters")
				printLinks(fromLinks[el], "Resources")
				printLinks(fromLinks[el], "Outputs")
			}
		}

		if twoWayTree && (allLinks || len(toLinks[el]) > 0) {
			if len(toLinks[el]) == 0 {
				fmt.Println("    UsedBy: []")
			} else {
				fmt.Println("    UsedBy:")
				printLinks(toLinks[el], "Parameters")
				printLinks(toLinks[el], "Resources")
				printLinks(toLinks[el], "Outputs")
			}
		}
	}
}

var dotShapes = map[string]string{
	"Parameters": "diamond",
	"Resources":  "Mrecord",
	"Outputs":    "rectangle",
}

func printDot(graph graph.Graph) {
	out := strings.Builder{}

	out.WriteString("digraph {\n")
	out.WriteString("    rankdir=LR;\n")
	out.WriteString("    concentrate=true;\n")

	// First pass, group types
	doGroup := func(t string) {
		out.WriteString(fmt.Sprintf("    subgraph cluster_%s {\n", t))
		out.WriteString(fmt.Sprintf("        label=\"%s\";\n", t))
		for _, el := range graph.Nodes() {
			if el.Type == t {
				nodeName := fmt.Sprintf("%s: %s", el.Type, el.Name)

				out.WriteString(fmt.Sprintf("        \"%s\" [label=\"%s\" shape=%s];\n", nodeName, el.Name, dotShapes[el.Type]))
			}
		}
		out.WriteString("    }\n")
		out.WriteString("\n")
	}

	doGroup("Parameters")
	doGroup("Resources")
	doGroup("Outputs")

	for _, from := range graph.Nodes() {
		fromStr := fmt.Sprintf("%s: %s", from.Type, from.Name)

		for _, to := range graph.Get(from) {
			toStr := fmt.Sprintf("%s: %s", to.Type, to.Name)

			out.WriteString(fmt.Sprintf("    \"%s\" -> \"%s\";\n", toStr, fromStr))
		}
	}

	out.WriteString("}")

	fmt.Println(out.String())
}
