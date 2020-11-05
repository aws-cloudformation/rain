package node

import (
	"gopkg.in/yaml.v3"
)

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
