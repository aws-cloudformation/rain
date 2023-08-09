package simplify

import (
	"gopkg.in/yaml.v3"
)

type orderingMod struct {
	props    []string
	children map[string]orderingMod
}

var ordersMod = orderingMod{
	props: []string{
		"AWSTemplateFormatVersion",
		"Description",
		"Metadata",
		"Parameters",
		"Rules",
		"Mappings",
		"Conditions",
		"Transform",
		"Globals",
		"Resources",
		"Outputs",
	},
	children: map[string]orderingMod{
		"Transform": {
			props: []string{},
			children: map[string]orderingMod{
				"*": {
					props: []string{"Name", "Parameters"},
				},
			},
		},
		"Metadata": {
			props: []string{},
			children: map[string]orderingMod{
				"*": {
					props: []string{"Description"},
				},
			},
		},
		"Parameters": {
			props: []string{},
			children: map[string]orderingMod{
				"*": {
					props: []string{"Description", "Type", "AllowedValues", "Default"},
				},
			},
		},
		"Resources": {
			props: []string{},
			children: map[string]orderingMod{
				"*": {
					props: []string{"CreationPolicy", "DeletionPolicy", "UpdatePolicy", "UpdateReplacePolicy", "Type", "DependsOn", "Metadata", "Properties"},
				},
			},
		},
		"Outputs": {
			props: []string{},
			children: map[string]orderingMod{
				"*": {
					props: []string{"Description", "Value", "Export"},
				},
			},
		},
	},
}

func modifyTemplate(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.DocumentNode {
		node = node.Content[0]
	}

	return modifyNode(node, ordersMod)
}

type nodeModifyMap struct {
	keys   []*yaml.Node
	keyMap map[string]*yaml.Node
	values map[string]*yaml.Node
}

func toModifyNodeMap(node *yaml.Node) nodeModifyMap {
	out := nodeModifyMap{
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

func modifyNode(node *yaml.Node, order orderingMod) *yaml.Node {
	if node == nil {
		return nil
	}

	if node.Kind != yaml.MappingNode {
		return node
	}

	// Discover the keys and values
	nm := toModifyNodeMap(node)

	// Apply any sub-orderings
	for key, value := range nm.values {
		if subOrder, ok := order.children[key]; ok {
			nm.values[key] = modifyNode(value, subOrder)
		} else if subOrder, ok := order.children["*"]; ok {
			nm.values[key] = modifyNode(value, subOrder)
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
