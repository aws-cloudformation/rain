package node

import (
	"encoding/json"
	"fmt"

	"github.com/aws-cloudformation/rain/internal/config"
	"gopkg.in/yaml.v3"
)

// NodePair represents a !!map key-value pair
type NodePair struct {
	Key   *yaml.Node
	Value *yaml.Node
}

// Clone returns a copy of the provided node
func Clone(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	out := &yaml.Node{
		Kind:        node.Kind,
		Style:       node.Style,
		Tag:         node.Tag,
		Value:       node.Value,
		Anchor:      node.Anchor,
		Alias:       Clone(node.Alias),
		Content:     make([]*yaml.Node, len(node.Content)),
		HeadComment: node.HeadComment,
		LineComment: node.LineComment,
		FootComment: node.FootComment,
		Line:        node.Line,
		Column:      node.Column,
	}

	for i, child := range node.Content {
		out.Content[i] = Clone(child)
	}

	return out
}

// Returns the parent node of node, paired with its name if it's a map pair.
// If it is not a map pair, only NodePair.Value is not nil.
// (YAML maps are arrays with even indexes being names and odd indexes being values)
//
// node
//
//	Content
//	  0: Name
//	  1: Map
//	    Content
//	      0: a
//	      1: b
//
// In the above, if I want b's parent node pair, I get [Name, Map]
// This allows us to ask "what is the parent's name",
// which is useful for knowing the logical name of the resource a node belongs to
func GetParent(node *yaml.Node, rootNode *yaml.Node, priorNode *yaml.Node) NodePair {
	if node == rootNode {
		config.Debugf("getParent node and rootNode are the same")
		return NodePair{node, node}
	}

	var found *yaml.Node
	var before *yaml.Node
	var pair NodePair

	if rootNode.Kind == yaml.DocumentNode || rootNode.Kind == yaml.SequenceNode {
		for i, n := range rootNode.Content {
			if n == node {
				found = rootNode
				before = priorNode
				break
			}
			var prior *yaml.Node
			if i > 0 {
				prior = rootNode.Content[i-1]
			}
			pair = GetParent(node, n, prior)
			if pair.Value != nil {
				found = pair.Value
				before = pair.Key
				break
			}
		}
	} else if rootNode.Kind == yaml.MappingNode {
		for i := 0; i < len(rootNode.Content); i += 2 {
			n := rootNode.Content[i+1]
			if n == node {
				found = rootNode
				before = priorNode
				break
			}
			var prior *yaml.Node
			prior = rootNode.Content[i]
			pair = GetParent(node, n, prior)
			if pair.Value != nil {
				found = pair.Value
				before = pair.Key
				break
			}
		}
	}
	return NodePair{Key: before, Value: found}
}

// Convert a node to JSON
func ToJson(node *yaml.Node) string {
	j, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		return fmt.Sprintf("Failed to marshal node to json: %v:", err)
	}
	return string(j)
}
