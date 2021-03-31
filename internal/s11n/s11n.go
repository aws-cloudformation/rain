package s11n

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// GetMapValue returns the key and value nodes from node that matches key.
// if node is not a mapping node or the key does not exist, GetMapValue returns nil
func GetMapValue(node *yaml.Node, key string) (*yaml.Node, *yaml.Node) {
	if node.Kind != yaml.MappingNode {
		return nil, nil
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Value == key {
			return keyNode, valueNode
		}
	}

	return nil, nil
}

// GetPath returns the node by descending into map and array nodes for each element of path
func GetPath(node *yaml.Node, path []interface{}) (*yaml.Node, error) {
	if node.Kind == yaml.DocumentNode {
		return GetPath(node.Content[0], path)
	}

	if len(path) == 0 {
		return node, nil
	}

	next, path := path[0], path[1:]

	switch v := next.(type) {
	case string:
		_, value := GetMapValue(node, v)
		if value == nil {
			return nil, fmt.Errorf("Could not find map key: '%s'", v)
		}

		return GetPath(value, path)
	case int:
		if node.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("Attempt to index non-sequence node with '%d'", v)
		}

		return GetPath(node.Content[v], path)
	default:
		return nil, fmt.Errorf("Unexpected path entry '%#v'", next)
	}
}

func setPath(node *yaml.Node, path []interface{}, value *yaml.Node) error {
	if len(path) == 0 {
		*node = *value
		return nil
	}

	path, last := path[:len(path)-1], path[len(path)-1]

	node, err := GetPath(node, path)
	if err != nil {
		return err
	}

	switch v := last.(type) {
	case string:
		if node.Kind != yaml.MappingNode {
			return fmt.Errorf("Attempt to set index of non-mapping node with '%s'", v)
		}

		if _, mapValue := GetMapValue(node, v); mapValue != nil {
			*mapValue = *value
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
