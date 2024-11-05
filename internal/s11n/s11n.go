package s11n

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"gopkg.in/yaml.v3"
)

// GetMapValue returns the key and value nodes from node that matches key.
// if node is not a mapping node or the key does not exist, GetMapValue returns nil
func GetMapValue(n *yaml.Node, key string) (*yaml.Node, *yaml.Node, error) {
	if n == nil {
		return nil, nil, fmt.Errorf("node is nil for key %s", key)
	}

	if n.Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("kind is %v for key %s", n.Kind, key)
	}

	if len(n.Content)%2 != 0 {
		return nil, nil, fmt.Errorf("uneven length %v for key %s", len(n.Content), key)
	}

	for i := 0; i < len(n.Content); i += 2 {
		keyNode := n.Content[i]
		if len(n.Content) <= i+1 {
			config.Debugf("GetMapValue about to step over array at i=%v, n:\n%s",
				i, node.ToSJson(n))
		}
		valueNode := n.Content[i+1]

		if keyNode.Value == key {
			return keyNode, valueNode, nil
		}
	}

	return nil, nil, fmt.Errorf("key %s not found", key)
}

// GetValue tries to get a scalar value from a mapping node
// If anything goes wrong, it returns an empty string.
// Use GetMapValue if you need more control.
func GetValue(n *yaml.Node, name string) string {
	_, v, _ := GetMapValue(n, name)
	if v != nil && v.Kind == yaml.ScalarNode {
		return v.Value
	}
	return ""
}

// GetMap returns a map of the names and values in a MappingNode
// This is a shorthand version of calling GetMapValue and iterating over the results.
// One difference is that keys will no longer be in their original order,
// which matters for several use cases like processing Constants.
func GetMap(n *yaml.Node, name string) map[string]*yaml.Node {
	_, c, _ := GetMapValue(n, name)
	if c == nil {
		return nil
	}
	retval := make(map[string]*yaml.Node)

	for i := 0; i < len(c.Content); i += 2 {
		k := c.Content[i].Value
		v := c.Content[i+1]
		retval[k] = v
	}
	return retval
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
		_, value, _ := GetMapValue(node, v)
		if value == nil {
			return nil, fmt.Errorf("could not find map key: '%s'", v)
		}

		return GetPath(value, path)
	case int:
		if node.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("attempt to index non-sequence node with '%d'", v)
		}

		return GetPath(node.Content[v], path)
	default:
		return nil, fmt.Errorf("unexpected path entry '%#v'", next)
	}
}

/*
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
			return fmt.Errorf("attempt to set index of non-mapping node with '%s'", v)
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
			return fmt.Errorf("attempt to set index of non-sequence node with '%d'", v)
		}

		if v < len(node.Content) {
			node.Content[v] = value
		} else {
			node.Content = append(node.Content, value)
		}

		return nil
	default:
		return fmt.Errorf("unexpected path entry '%#v'", last)
	}
}
*/
