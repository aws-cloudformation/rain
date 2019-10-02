// Package cfn provides the Template type that models a CloudFormation template.
//
// The sub-packages of cfn contain various tools for working with templates
package cfn

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cfn/diff"
	"github.com/aws-cloudformation/rain/cfn/graph"
)

const pseudoParameterType = "Parameter"

// Element represents a top-level entry in a CloudFormation template
// for example a resource, parameter, or output
type Element struct {
	// Name is the name of the element
	Name string

	// Type is the name of the top-level part of a CloudFormation
	// that contains this Element (e.g. Resources, Parameters)
	Type string
}

// Template represents a CloudFormation template. The Template type
// is minimal for now but will likely grow new features as needed by rain.
type Template map[string]interface{}

// Map returns the template as a map[string]interface{}
// This can be used for easy serialisation to e.g. JSON or YAML
func (t Template) Map() map[string]interface{} {
	return map[string]interface{}(t)
}

// Diff returns a Diff object representing the difference
// between this template and the template passed to Diff
func (t Template) Diff(other Template) diff.Diff {
	return diff.New(t.Map(), other.Map())
}

// Graph returns a Graph representing the connections
// between elements in the template.
// The type of each item in the graph should be Element
func (t Template) Graph() graph.Graph {
	// Map out parameter and resource names so we know which is which
	entities := make(map[string]string)
	for typeName, entity := range t.Map() {
		if typeName != "Parameters" && typeName != "Resources" {
			continue
		}

		if entityTree, ok := entity.(map[string]interface{}); ok {
			for entityName, _ := range entityTree {
				entities[entityName] = typeName
			}
		}
	}

	// Now find the deps
	graph := graph.New()
	for typeName, entity := range t.Map() {
		if typeName != "Resources" && typeName != "Outputs" {
			continue
		}

		if entityTree, ok := entity.(map[string]interface{}); ok {
			for fromName, res := range entityTree {
				from := Element{fromName, typeName}
				graph.Add(from)

				resource := res.(map[string]interface{})
				for _, toName := range getRefs(resource) {
					toName = strings.Split(toName, ".")[0]

					toType, ok := entities[toName]

					if !ok {
						if strings.HasPrefix(toName, "AWS::") {
							toType = "Parameters"
						} else {
							panic(fmt.Sprintf("Template has unresolved dependency '%s' at %s: %s", toName, typeName, fromName))
						}
					}

					graph.Add(from, Element{toName, toType})
				}
			}
		}
	}

	return graph
}
