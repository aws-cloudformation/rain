// ForEach (aka Map) processing for modules
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
	MI = "$Index"
	MV = "$Identifier"
)

func replaceMapStr(s string, index int, key string, identifier string) string {
	s = strings.Replace(s, "${"+MI+"}", fmt.Sprintf("%d", index), -1)
	s = strings.Replace(s, "&{"+MI+"}", fmt.Sprintf("%d", index), -1)
	s = strings.Replace(s, MI, fmt.Sprintf("%d", index), -1)
	s = strings.Replace(s, "${"+MV+"}", key, -1)
	s = strings.Replace(s, "&{"+MV+"}", key, -1)
	s = strings.Replace(s, MV, key, -1)

	if identifier != "" {
		s = cft.ReplaceIdentifier(s, key, identifier)
	}

	return s
}

// mapPlaceholders looks for Index and Identifier and replaces them
func mapPlaceholders(n *yaml.Node, index int, key string, identifier string) {

	vf := func(v *visitor.Visitor) {
		yamlNode := v.GetYamlNode()
		if yamlNode.Kind == yaml.MappingNode {
			content := yamlNode.Content
			if len(content) == 2 {
				switch content[0].Value {
				case string(cft.Sub):
					r := replaceMapStr(content[1].Value,
						index, key, identifier)
					if parse.IsSubNeeded(r) {
						yamlNode.Value = r
					} else {
						*yamlNode = *node.MakeScalar(r)
					}
				case string(cft.GetAtt):
					for _, getatt := range content[1].Content {
						getatt.Value = replaceMapStr(getatt.Value, index, key,
							identifier)
					}
				}
			}
		} else if yamlNode.Kind == yaml.ScalarNode {
			yamlNode.Value = replaceMapStr(yamlNode.Value, index, key,
				identifier)
		}
	}

	visitor.NewVisitor(n).Visit(vf)
}

// getMapKeys gets the CSV key values from either a hard-coded
// string or from a Ref.
func getMapKeys(moduleConfig *cft.ModuleConfig, t *cft.Template,
	parentModule *Module) ([]string, error) {

	// The map is either a CSV or a Ref to a CSV that we can fully
	// resolve
	mapJson := node.ToSJson(moduleConfig.Map)
	var keys []string
	if moduleConfig.Map.Kind == yaml.ScalarNode {
		keys = node.StringsFromNode(moduleConfig.Map)
	} else if moduleConfig.Map.Kind == yaml.SequenceNode {
		keys = node.StringsFromNode(moduleConfig.Map)
	} else if moduleConfig.Map.Kind == yaml.MappingNode {
		if cft.IsRef(moduleConfig.Map) {
			r := moduleConfig.Map.Content[1].Value
			// Look in the parent templates Parameters for the Ref
			if !t.HasSection(cft.Parameters) {
				msg := "module Map Ref no Parameters: %s"
				return nil, fmt.Errorf(msg, mapJson)
			}
			params, _ := t.GetSection(cft.Parameters)
			_, keysNode, _ := s11n.GetMapValue(params, r)
			if keysNode == nil {
				msg := "expected module Map Ref to a Parameter: %s"
				return nil, fmt.Errorf(msg, mapJson)
			}

			// Look at the parent module Properties
			// TODO: Will this work in nested modules?
			// Have those props been resolved?
			if parentModule != nil {
				if parentVal, ok := parentModule.Config.Properties()[r]; ok {
					csv, ok := parentVal.(string)
					if ok {
						keys = strings.Split(csv, ",")
					} else {
						msg := "expected Map keys to be a CSV: %v"
						return nil, fmt.Errorf(msg, parentVal)
					}
				}
			}

			if len(keys) == 0 {
				_, d, _ := s11n.GetMapValue(keysNode, "Default")
				if d == nil {
					msg := "expected module Map Ref to a Default: %s"
					return nil, fmt.Errorf(msg, mapJson)
				}
				keys = node.StringsFromNode(d)
			}
		} else {
			msg := "expected module Map to be a Ref: %s"
			return nil, fmt.Errorf(msg, mapJson)
		}
	} else {
		return nil, fmt.Errorf("unexpected module Map Kind: %s", mapJson)
	}

	return keys, nil
}

// processMaps duplicates module configuration in the template for
// each value in a CSV. The external name for this is now "ForEach",
// but originally it was called "Map".
func processMaps(originalContent []*yaml.Node,
	t *cft.Template, parentModule *Module) ([]*yaml.Node, error) {

	content := make([]*yaml.Node, 0)

	// Process Maps, which duplicate the module for each element in a list
	for i := 0; i < len(originalContent); i += 2 {
		name := originalContent[i].Value
		moduleConfig, err := t.ParseModuleConfig(name, originalContent[i+1])
		if err != nil {
			return nil, err
		}

		if moduleConfig.Map == nil {
			content = append(content, originalContent[i])
			content = append(content, originalContent[i+1])
			continue
		}

		keys, err := getMapKeys(moduleConfig, t, parentModule)
		if err != nil {
			return nil, err
		}

		if len(keys) < 1 {
			msg := "expected module Map to have items: %s"
			mapErr := fmt.Errorf(msg, node.YamlStr(moduleConfig.Node))
			return nil, mapErr
		}

		// Duplicate the config
		for i, key := range keys {
			mapName := fmt.Sprintf("%s%d", moduleConfig.Name, i)

			if moduleConfig.FnForEach != nil &&
				moduleConfig.FnForEach.OutputKeyHasIdentifier() {
				// The OutputKey is something like A${Identifier},
				// which means we use the key instead of the
				// array index to create the logical id
				mapName = fmt.Sprintf("%s%s", moduleConfig.Name, key)
			}

			copiedNode := node.Clone(moduleConfig.Node)
			node.RemoveFromMap(copiedNode, ForEach)
			copiedConfig, err := t.ParseModuleConfig(mapName, copiedNode)
			if err != nil {
				return nil, err
			}

			// These values won't go into the YAML but we'll store them for
			// later

			copiedConfig.OriginalName = moduleConfig.Name
			copiedConfig.IsMapCopy = true
			copiedConfig.MapIndex = i
			copiedConfig.MapKey = key

			// Add a reference to the template so we can find it later for
			// Outputs

			t.AddMappedModule(copiedConfig)

			// Replace $Index and $Identifier
			identifier := ""
			if moduleConfig.FnForEach != nil {
				identifier = moduleConfig.FnForEach.Identifier
			}
			mapPlaceholders(copiedNode, i, key, identifier)

			content = append(content, node.MakeScalar(mapName))
			content = append(content, copiedNode)
		}
	}
	return content, nil
}
