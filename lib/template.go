package lib // FIXME: Rename this

import (
	"fmt"
	"regexp"
	"strings"
)

const pseudoParameterType = "PseudoParameter"

type Template map[string]interface{}
type tree map[string]interface{}

var typeMap = map[string]string{
	"Resources":  "Resource",
	"Outputs":    "Output",
	"Parameters": "Parameter",
}

var subRe = regexp.MustCompile(`\$\{([^!].+?)\}`)

type Node struct {
	Type string
	Name string
}

type Dependency struct {
	From Node
	To   Node
}

func (n Node) String() string {
	return fmt.Sprintf("%s / %s", n.Type, n.Name)
}

func (d Dependency) String() string {
	return fmt.Sprintf("%s -> %s", d.From, d.To)
}

func (t Template) FindDependencies() []Dependency {
	deps := make([]Dependency, 0)

	var entityTypes = make(map[string]string)

	for typeName, entity := range t {
		entityTree, ok := entity.(map[string]interface{})

		if ok {
			for entityName, _ := range entityTree {
				entityTypes[entityName] = typeMap[typeName]
			}
		}
	}

	for typeKey, typeName := range typeMap {
		if _, ok := t[typeKey]; !ok {
			continue
		}

		// Resources
		for name, res := range t[typeKey].(map[string]interface{}) {
			resource := tree(res.(map[string]interface{}))
			for to := range resource.Refs() {
				toType, ok := entityTypes[to]
				if !ok {
					switch {
					case strings.HasPrefix(to, "AWS::"):
						toType = pseudoParameterType
					default:
						toType = "Unknown"
					}
				}

				deps = append(deps, Dependency{
					From: Node{typeName, name},
					To:   Node{toType, to}, // FIXME: Find out what type of thing this is
				})
			}
		}
	}

	return deps
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
