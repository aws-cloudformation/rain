package visitor

import (
	"gopkg.in/yaml.v3"
)

type Visitor struct {
	rootNode   *yaml.Node
	stop       bool
	skip       bool
	parentNode *yaml.Node
}

type FilterFunc func(*Visitor) bool
type VisitFunc func(*Visitor)

func (v *Visitor) GetYamlNode() *yaml.Node {
	return v.rootNode
}

func (v *Visitor) GetParentNode() *yaml.Node {
	return v.parentNode
}

func (v *Visitor) SkipChildren() {
	v.skip = true
}

// Stop can be called from a visitor function to stop recursion
func (v *Visitor) Stop() {
	v.stop = true
}

func NewVisitor(root *yaml.Node) *Visitor {
	return &Visitor{
		rootNode: root,
	}
}

func (v *Visitor) Visit(visitFunc VisitFunc) {
	var walk VisitFunc
	walk = func(node *Visitor) {
		visitFunc(node)
		if node.stop {
			return
		}
		if !node.skip {
			for _, child := range node.rootNode.Content {
				childVisitor := NewVisitor(child)
				childVisitor.parentNode = node.rootNode
				walk(childVisitor)
				if childVisitor.stop {
					return
				}
			}
		}
	}
	node := NewVisitor(v.rootNode)
	walk(node)
}

func (v *Visitor) Match(filterFunc FilterFunc) []*yaml.Node {
	var results []*yaml.Node
	v.Visit(func(node *Visitor) {
		if filterFunc(node) {
			results = append(results, node.rootNode)
		}
	})
	return results
}
