// Package graph provides functionality to build
// a graph of connected nodes with a cfn.Template
package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/config"
)

// Node represents a top-level entry in a CloudFormation template
// for example a resource, parameter, or output
type Node struct {
	// Type is the name of the top-level part of a template
	// that contains this Node (e.g. Resources, Parameters)
	Type string

	// Name is the name of the Node
	Name string
}

func (n Node) String() string {
	return fmt.Sprintf("%s/%s", n.Type, n.Name)
}

// Graph represents a directed, acyclic graph with ordered nodes
type Graph struct {
	nodes map[Node]map[Node]bool
	order []Node
}

// New returns a Graph representing the connections
// between elements in the provided template.
// The type of each item in the graph is Node
func New(t cft.Template) Graph {
	// Map out parameter and resource names so we know which is which
	entities := make(map[string]string)
	for typeName, entity := range t.Map() {
		if typeName != "Parameters" && typeName != "Resources" {
			continue
		}

		if entityTree, ok := entity.(map[string]interface{}); ok {
			for entityName := range entityTree {
				entities[entityName] = typeName
			}
		}
	}

	// Now find the deps
	graph := Graph{
		nodes: make(map[Node]map[Node]bool),
		order: make([]Node, 0),
	}

	for typeName, entity := range t.Map() {
		if typeName != "Resources" && typeName != "Outputs" {
			continue
		}

		if entityTree, ok := entity.(map[string]interface{}); ok {
			for fromName, res := range entityTree {
				from := Node{typeName, fromName}
				graph.link(from)

				resource := res.(map[string]interface{})
				for _, toName := range getRefs(resource) {
					toName = strings.Split(toName, ".")[0]

					toType, ok := entities[toName]

					if !ok {
						if strings.HasPrefix(toName, "AWS::") {
							toType = "Parameters"
						} else {
							config.Debugf("template has unresolved dependency '%s' at %s: %s", toName, typeName, fromName)
							continue
						}
					}

					graph.link(from, Node{toType, toName})
				}
			}
		}
	}

	return graph
}

func (g *Graph) String() string {
	out := strings.Builder{}

	for _, left := range g.order {
		out.WriteString(fmt.Sprintf("%s:\n", left))
		if len(g.nodes[left]) > 0 {
			for right := range g.nodes[left] {
				out.WriteString(fmt.Sprintf("- %s\n", right))
			}
		}
	}

	return out.String()
}

func (g *Graph) add(item Node) {
	if _, ok := g.nodes[item]; !ok {
		g.nodes[item] = make(map[Node]bool)
		g.order = append(g.order, item)
	}
}

// link creates a connection between two nodes in the graph
func (g *Graph) link(item Node, links ...Node) {
	g.add(item)

	for _, to := range links {
		g.add(to)
		g.nodes[item][to] = true
	}
}

func (g Graph) depth(item Node) int {
	seen := map[Node]bool{
		item: true,
	}

	count := 0

	var dive func(Node)

	dive = func(from Node) {
		for to := range g.nodes[from] {
			if !seen[to] {
				seen[to] = true
				count++

				dive(to)
			}
		}
	}

	dive(item)

	return count
}

// Nodes returns all nodes of the graph, in order of their dependencies.
// Nodes with the fewest dependencies are at the beginning of the slice.
func (g Graph) Nodes() []Node {
	sort.Slice(g.order, func(i, j int) bool {
		a, b := g.order[i], g.order[j]

		diff := g.depth(a) - g.depth(b)

		if diff == 0 {
			return fmt.Sprint(a) < fmt.Sprint(b)
		}

		return diff < 0
	})

	return g.order
}

// Get returns all nodes that are connected to the item that you pass in.
func (g Graph) Get(item Node) []Node {
	links := make([]Node, 0)
	for to := range g.nodes[item] {
		links = append(links, to)
	}

	sort.Slice(links, func(i, j int) bool {
		return fmt.Sprint(links[i]) < fmt.Sprint(links[j])
	})

	return links
}

// GetReverse returns all nodes that connect to the item that you pass in.
func (g Graph) GetReverse(item Node) []Node {
	links := make([]Node, 0)
	for from, deps := range g.nodes {
		if _, ok := deps[item]; ok {
			links = append(links, from)
		}
	}

	sort.Slice(links, func(i, j int) bool {
		return fmt.Sprint(links[i]) < fmt.Sprint(links[j])
	})

	return links
}
