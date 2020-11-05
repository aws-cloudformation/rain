package format

import (
	"encoding/json"
	"strings"

	"github.com/mickep76/mapslice-json"
	"gopkg.in/yaml.v3"
)

func handleScalar(node *yaml.Node) interface{} {
	if node.Tag != "" && !strings.HasPrefix(node.Tag, "!!") {
		// Convert back to map style
		tag := strings.TrimPrefix(node.Tag, "!")

		if tag != "Ref" {
			tag = "Fn::" + tag
		}

		return mapslice.MapSlice{
			mapslice.MapItem{
				Key:   tag,
				Value: node.Value,
			},
		}
	}

	// Marshal and unmarshal the node and return the result
	intermediate, err := yaml.Marshal(node)
	if err != nil {
		panic(err)
	}

	var out interface{}
	err = yaml.Unmarshal(intermediate, &out)
	if err != nil {
		panic(err)
	}

	return out
}

func jsonise(node *yaml.Node) interface{} {
	switch node.Kind {
	case yaml.DocumentNode:
		return jsonise(node.Content[0])
	case yaml.MappingNode:
		out := make(mapslice.MapSlice, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			out[i/2] = mapslice.MapItem{
				Key:   jsonise(key),
				Value: jsonise(value),
			}
		}
		return out
	case yaml.SequenceNode:
		out := make([]interface{}, len(node.Content))
		for i, n := range node.Content {
			out[i] = jsonise(n)
		}
		return out
	default:
		return handleScalar(node)
	}
}

func convertToJSON(in string) string {
	var d yaml.Node
	err := yaml.Unmarshal([]byte(in), &d)
	if err != nil {
		panic(err)
	}

	intermediate := jsonise(&d)

	out, err := json.MarshalIndent(&intermediate, "", "    ")
	if err != nil {
		panic(err)
	}
	return string(out)
}
