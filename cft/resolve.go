package cft

import (
	"strings"

	"gopkg.in/yaml.v3"
)

type resolveFunc func(*yaml.Node, map[string]string)

var resolvers map[string]resolveFunc

func init() {
	resolvers = map[string]resolveFunc{
		"Ref":     resolveRef,
		"Fn::Sub": resolveSub,
	}
}

func resolve(node *yaml.Node, params map[string]string) {
	if node.Kind == yaml.MappingNode {
		if len(node.Content) == 2 {
			key := node.Content[0].Value

			if resolver, ok := resolvers[key]; ok {
				resolver(node, params)
			}
		}
	}

	// Map children
	if node.Kind == yaml.MappingNode {
		for i := 1; i < len(node.Content); i += 2 {
			resolve(node.Content[i], params)
		}
	} else if node.Kind == yaml.SequenceNode || node.Kind == yaml.DocumentNode {
		for _, child := range node.Content {
			resolve(child, params)
		}
	}
}

func resolveRef(node *yaml.Node, params map[string]string) {
	content := node.Content[1]

	if content.Kind == yaml.ScalarNode {
		if value, ok := params[content.Value]; ok {
			// Explicitly ignoring the error
			node.Encode(value)
		}
	}
}

func resolveSub(node *yaml.Node, params map[string]string) {
	content := node.Content[1]
	s := content
	values := make(map[string]string)
	for key, value := range params {
		values[key] = value
	}

	if content.Kind == yaml.SequenceNode {
		if len(content.Content) > 0 {
			s = content.Content[0]
		}

		if len(content.Content) == 2 {
			vars := content.Content[1]

			if vars.Kind == yaml.MappingNode {
				for i := 0; i < len(vars.Content); i += 2 {
					key := vars.Content[i]
					value := vars.Content[i+1]

					resolve(value, params)

					values[key.Value] = value.Value
				}
			}
		}
	}

	out := s.Value
	for key, value := range values {
		out = strings.ReplaceAll(out, "${"+key+"}", value)
	}

	node.Encode(out)
}
