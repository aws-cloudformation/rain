package cft

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type mapNode struct {
	key   *yaml.Node
	value *yaml.Node
}

func getMapNode(node *yaml.Node, key string) (*mapNode, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("Attempt to index non-mapping node with '%s'", key)
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Value == key {
			return &mapNode{keyNode, valueNode}, nil
		}
	}

	return nil, fmt.Errorf("Key not found '%s'", key)
}

func getNodePath(node *yaml.Node, path []interface{}) (*yaml.Node, error) {
	if node.Kind == yaml.DocumentNode {
		return getNodePath(node.Content[0], path)
	}

	if len(path) == 0 {
		return node, nil
	}

	next, path := path[0], path[1:]

	switch v := next.(type) {
	case string:
		kvp, err := getMapNode(node, v)
		if err != nil {
			return nil, err
		}

		return getNodePath(kvp.value, path)
	case int:
		if node.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("Attempt to index non-sequence node with '%d'", v)
		}

		return getNodePath(node.Content[v], path)
	default:
		return nil, fmt.Errorf("Unexpected path entry '%#v'", next)
	}
}

func setNodePath(node *yaml.Node, path []interface{}, value *yaml.Node) error {
	if len(path) == 0 {
		*node = *value
		return nil
	}

	path, last := path[:len(path)-1], path[len(path)-1]

	node, err := getNodePath(node, path)
	if err != nil {
		return err
	}

	switch v := last.(type) {
	case string:
		if node.Kind != yaml.MappingNode {
			return fmt.Errorf("Attempt to set index of non-mapping node with '%s'", v)
		}

		if kvp, err := getMapNode(node, v); err == nil {
			kvp.value = value
		} else {
			node.Content = append(node.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: v,
			}, value)
		}

		return nil
	case int:
		if node.Kind != yaml.SequenceNode {
			return fmt.Errorf("Attempt to set index of non-sequence node with '%d'", v)
		}

		if v < len(node.Content) {
			node.Content[v] = value
		} else {
			node.Content = append(node.Content, value)
		}

		return nil
	default:
		return fmt.Errorf("Unexpected path entry '%#v'", last)
	}
}
