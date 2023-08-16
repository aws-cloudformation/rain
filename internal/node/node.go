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

	// Parent is used by modules to reference the parent template resource
	Parent *NodePair
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
		return NodePair{Key: node, Value: node}
	}

	if node == nil {
		config.Debugf("node is nil")
		return NodePair{nil, nil, nil}
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
			pair = GetParent(node, n, rootNode.Content[i])
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

// Remove a map key-value pair from node.Content
func RemoveFromMap(node *yaml.Node, name string) error {

	if len(node.Content) == 0 {
		return nil
	}

	idx := -1

	for i, n := range node.Content {
		if n.Value == name {
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("unable to remove %v from map", name)
	}

	newContent := make([]*yaml.Node, 0)
	newContent = append(newContent, node.Content[:idx]...)
	// Remove 2 elements, since the key and value are consecutive elements in the Content array
	newContent = append(newContent, node.Content[idx+2:]...)

	node.Content = newContent

	return nil
}

// Add or replace a map value
func SetMapValue(parent *yaml.Node, name string, val *yaml.Node) {
	found := false
	for i, v := range parent.Content {
		if v.Kind == yaml.ScalarNode && v.Value == name {
			found = true
			parent.Content[i+1] = val
			break
		}
	}
	if !found {
		parent.Content = append(parent.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: name})
		parent.Content = append(parent.Content, val)
	}
}
