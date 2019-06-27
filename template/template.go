package template

import (
	"fmt"
	"regexp"
	"strings"
)

type Template map[string]interface{}
type tree map[string]interface{}

type Element struct {
	Name string
	Type string
}

const pseudoParameterType = "Parameter"

var subRe = regexp.MustCompile(`\$\{([^!].+?)\}`)

func (t Template) Graph() Graph {
	// Map out parameter and resource names so we know which is which
	entities := make(map[string]string)
	for typeName, entity := range t {
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
	graph := NewGraph()
	for typeName, entity := range t {
		if typeName != "Resources" && typeName != "Outputs" {
			continue
		}

		if entityTree, ok := entity.(map[string]interface{}); ok {
			for fromName, res := range entityTree {
				from := Element{fromName, typeName}
				graph.Add(from)

				resource := tree(res.(map[string]interface{}))
				for toName := range resource.Refs() {
					toType, ok := entities[toName]
					if !ok {
						if strings.HasPrefix(toName, "AWS::") {
							toType = "Parameters"
						} else {
							panic(fmt.Errorf("Template has unresolved dependency '%s' at %s: %s", toName, typeName, fromName))
						}
					}

					graph.Link(from, Element{toName, toType})
				}
			}
		}
	}

	return graph
}

func (t tree) Refs() chan string {
	ch := make(chan string)

	go func() {
		defer close(ch)
		t.findRefs(ch)
	}()

	return ch
}

func (t tree) findRefs(ch chan string) {
	for key, value := range t {
		switch key {
		case "Ref":
			ch <- value.(string)
		case "Fn::GetAtt":
			switch v := value.(type) {
			case string:
				parts := strings.Split(v, ".")
				ch <- parts[0]
			case []interface{}:
				if s, ok := v[0].(string); ok {
					ch <- s
				}
			default:
				fmt.Printf("Malformed GetAtt: %T\n", v)
			}
		case "Fn::Sub":
			switch v := value.(type) {
			case string:
				for _, groups := range subRe.FindAllStringSubmatch(v, 1) {
					ch <- groups[1]
				}
			case []interface{}:
				if parts, ok := v[1].(tree); ok {
					for _, part := range parts {
						ch <- part.(string)
					}
				}
			default:
				fmt.Printf("Malformed Sub: %T\n", v)
			}
		default:
			treeChan := make(chan tree)

			go func() {
				findTrees(treeChan, value)
				close(treeChan)
			}()

			for tree := range treeChan {
				tree.findRefs(ch)
			}
		}
	}
}

func findTrees(ch chan tree, value interface{}) {
	switch v := value.(type) {
	case map[string]interface{}:
		ch <- tree(v)
	case []interface{}:
		for _, child := range v {
			findTrees(ch, child)
		}
	}
}
