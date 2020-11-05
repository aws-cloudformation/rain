package cft

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Comment represents a path to a node and a comment string to attach to it
type Comment struct {
	Path  []interface{}
	Value string
}

// AddComments applies a set of comments to the template
func (t Template) AddComments(comments []*Comment) error {
	node := &t.Node

	if node.Kind == yaml.DocumentNode {
		node = node.Content[0]
	}

	for _, comment := range comments {
		var n *yaml.Node
		var err error

		if len(comment.Path) == 0 {
			n, err = getNodePath(node, comment.Path)
			if err != nil {
				return err
			}

			if len(n.Content) == 0 {
				return fmt.Errorf("Unable to set head at node root")
			}

			n.Content[0].HeadComment = comment.Value

			continue
		}

		path, last := comment.Path[0:len(comment.Path)-1], comment.Path[len(comment.Path)-1]

		n, err = getNodePath(node, path)
		if err != nil {
			return err
		}

		switch v := last.(type) {
		case string:
			kvp, err := getMapNode(n, v)
			if err != nil {
				return err
			}

			switch kvp.value.Kind {
			case yaml.MappingNode, yaml.SequenceNode:
				if len(kvp.value.Content) == 0 {
					kvp.value.LineComment = comment.Value
				} else {
					kvp.key.LineComment = comment.Value
				}
			default:
				kvp.value.LineComment = comment.Value
			}
		case int:
			n, err = getNodePath(node, comment.Path)
			if err != nil {
				return err
			}

			if n.Kind == yaml.ScalarNode {
				n.LineComment = comment.Value
			} else {
				n.HeadComment = comment.Value
			}
		default:
			return fmt.Errorf("Unexpected path element '%#v'", v)
		}
	}

	return nil
}
