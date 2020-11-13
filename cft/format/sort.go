package format

import (
	"gopkg.in/yaml.v3"
)

type ordering struct {
	props    []string
	children map[string]ordering
}

var orders = ordering{
	props: []string{
		"AWSTemplateFormatVersion",
		"Description",
		"Metadata",
		"Parameters",
		"Mappings",
		"Conditions",
		"Transform",
		"Globals",
		"Resources",
		"Outputs",
	},
	children: map[string]ordering{
		"Transform": {
			props: []string{},
			children: map[string]ordering{
				"*": {
					props: []string{"Name", "Parameters"},
				},
			},
		},
		"Metadata": {
			props: []string{},
			children: map[string]ordering{
				"*": {
					props: []string{"Description"},
				},
			},
		},
		"Parameters": {
			props: []string{},
			children: map[string]ordering{
				"*": {
					props: []string{"Description", "Type", "AllowedValues", "Default"},
				},
			},
		},
		"Resources": {
			props: []string{},
			children: map[string]ordering{
				"*": {
					props: []string{"CreationPolicy", "DeletionPolicy", "UpdatePolicy", "UpdateReplacePolicy", "Type", "DependsOn", "Metadata", "Properties"},
				},
			},
		},
		"Outputs": {
			props: []string{},
			children: map[string]ordering{
				"*": {
					props: []string{"Description", "Value", "Export"},
				},
			},
		},
	},
}

func orderTemplate(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.DocumentNode {
		node = node.Content[0]
	}

	return orderNode(node, orders)
}

type nodeMap struct {
	keys   []*yaml.Node
	keyMap map[string]*yaml.Node
	values map[string]*yaml.Node
}

func toNodeMap(node *yaml.Node) nodeMap {
	out := nodeMap{
		keys:   make([]*yaml.Node, 0),
		keyMap: make(map[string]*yaml.Node),
		values: make(map[string]*yaml.Node),
	}

	if node.Kind != yaml.MappingNode {
		return out
	}

	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		value := node.Content[i+1]

		out.keys = append(out.keys, key)
		out.keyMap[key.Value] = key
		out.values[key.Value] = value
	}

	return out
}

func orderNode(node *yaml.Node, order ordering) *yaml.Node {
	if node == nil {
		return nil
	}

	if node.Kind != yaml.MappingNode {
		return node
	}

	// Discover the keys and values
	nm := toNodeMap(node)

	// Apply any sub-orderings
	for key, value := range nm.values {
		if subOrder, ok := order.children[key]; ok {
			nm.values[key] = orderNode(value, subOrder)
		} else if subOrder, ok := order.children["*"]; ok {
			nm.values[key] = orderNode(value, subOrder)
		}
	}

	// Build the output node
	out := &yaml.Node{
		Kind: yaml.MappingNode,
	}

	done := make(map[string]bool)

	// Place priority keys first
	for _, prop := range order.props {
		if child, ok := nm.values[prop]; ok {
			out.Content = append(out.Content, nm.keyMap[prop])
			out.Content = append(out.Content, child)
			done[prop] = true
		}
	}

	// Place remaining keys
	for _, key := range nm.keys {
		if !done[key.Value] {
			out.Content = append(out.Content, key)
			out.Content = append(out.Content, nm.values[key.Value])
		}
	}

	return out
}
