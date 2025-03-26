package pkg

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/cft/visitor"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

const (
	MI = "$MapIndex"
	MV = "$MapValue"
)

func replaceMapStr(s string, index int, key string) string {
	s = strings.Replace(s, MI, fmt.Sprintf("%d", index), -1)
	s = strings.Replace(s, MV, key, -1)
	return s
}

// mapPlaceholders looks for MapValue and MapIndex and replaces them
func mapPlaceholders(n *yaml.Node, index int, key string) {

	vf := func(v *visitor.Visitor) {
		yamlNode := v.GetYamlNode()
		if yamlNode.Kind == yaml.MappingNode {
			content := yamlNode.Content
			if len(content) == 2 {
				switch content[0].Value {
				case string(cft.Sub):
					r := replaceMapStr(content[1].Value, index, key)
					if parse.IsSubNeeded(r) {
						yamlNode.Value = r
					} else {
						*yamlNode = *node.MakeScalar(r)
					}
				case string(cft.GetAtt):
					for _, getatt := range content[1].Content {
						getatt.Value = replaceMapStr(getatt.Value, index, key)
					}
				}
			}
		} else if yamlNode.Kind == yaml.ScalarNode {
			yamlNode.Value = replaceMapStr(yamlNode.Value, index, key)
		}
	}

	visitor.NewVisitor(n).Visit(vf)
}

func processMaps(originalContent []*yaml.Node, t *cft.Template) ([]*yaml.Node, error) {
	content := make([]*yaml.Node, 0)

	// This will hold info about original mapped modules
	moduleMaps := make(map[string]any)

	// Process Maps, which duplicate the module for each element in a list
	for i := 0; i < len(originalContent); i += 2 {
		name := originalContent[i].Value
		moduleConfig, err := cft.ParseModuleConfig(name, originalContent[i+1])
		if err != nil {
			return nil, err
		}

		if moduleConfig.Map != nil {
			// The map is either a CSV or a Ref to a CSV that we can fully resolve
			mapJson := node.ToSJson(moduleConfig.Map)
			var keys []string
			if moduleConfig.Map.Kind == yaml.ScalarNode {
				keys = node.StringsFromNode(moduleConfig.Map)
			} else if moduleConfig.Map.Kind == yaml.SequenceNode {
				keys = node.StringsFromNode(moduleConfig.Map)
			} else if moduleConfig.Map.Kind == yaml.MappingNode {
				if len(moduleConfig.Map.Content) == 2 && moduleConfig.Map.Content[0].Value == "Ref" {
					r := moduleConfig.Map.Content[1].Value
					// Look in the parent templates Parameters for the Ref
					if !t.HasSection(cft.Parameters) {
						return nil, fmt.Errorf("module Map Ref no Parameters: %s", mapJson)
					}
					params, _ := t.GetSection(cft.Parameters)
					_, keysNode, _ := s11n.GetMapValue(params, r)
					if keysNode == nil {
						return nil, fmt.Errorf("expected module Map Ref to a Parameter: %s", mapJson)
					}
					// TODO - This is too simple. What about sub-modules?
					_, d, _ := s11n.GetMapValue(keysNode, "Default")
					if d == nil {
						return nil, fmt.Errorf("expected module Map Ref to a Default: %s", mapJson)
					}
					keys = node.StringsFromNode(d)
				} else {
					return nil, fmt.Errorf("expected module Map to be a Ref: %s", mapJson)
				}
			} else {
				return nil, fmt.Errorf("unexpected module Map Kind: %s", mapJson)
			}

			if len(keys) < 1 {
				mapErr := fmt.Errorf("expected module Map to have items: %s", mapJson)
				return nil, mapErr
			}

			// Record the number of items in the map
			moduleMaps[moduleConfig.Name] = len(keys)

			// Duplicate the config
			for i, key := range keys {
				mapName := fmt.Sprintf("%s%d", moduleConfig.Name, i)
				copiedNode := node.Clone(moduleConfig.Node)
				node.RemoveFromMap(copiedNode, "Map")
				copiedConfig, err := cft.ParseModuleConfig(mapName, copiedNode)
				if err != nil {
					return nil, err
				}

				// These values won't go into the YAML but we'll store them for later
				copiedConfig.OriginalName = moduleConfig.Name
				copiedConfig.IsMapCopy = true
				copiedConfig.MapIndex = i
				copiedConfig.MapKey = key

				// Add a reference to the template so we can find it later for Outputs
				t.AddMappedModule(copiedConfig)

				// Replace $MapIndex and $MapValue
				mapPlaceholders(copiedNode, i, key)

				content = append(content, node.MakeScalar(mapName))
				content = append(content, copiedNode)
			}
		} else {
			content = append(content, originalContent[i])
			content = append(content, originalContent[i+1])
		}
	}
	return content, nil
}
