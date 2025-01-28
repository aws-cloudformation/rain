package merge

import (
	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/visitor"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// mergeOutputImports looks for exported output values that are imported in this template.
// This happens when we merge template A that has Outputs, and template B imports them.
// Since we are merging the templates, the outputs and imports are not necessary.
// If there are any, it replaces the Fn::ImportValue nodes with the value.
// Otherwise the template is returned as is
func mergeOutputImports(t cft.Template) (cft.Template, error) {
	outputs, err := t.GetSection(cft.Outputs)
	if err != nil {
		// This is expected if the template has no Outputs
		config.Debugf("mergeOutputImports has no outputs: %v", err)
		return t, nil
	}
	exportMap := make(map[string]*yaml.Node)
	for i := 0; i < len(outputs.Content); i += 2 {
		name := outputs.Content[i].Value
		val := outputs.Content[i+1]

		config.Debugf("Checking %s: %s", name, node.ToSJson(val))

		_, exp, _ := s11n.GetMapValue(val, "Export")
		if exp != nil {
			_, exportName, _ := s11n.GetMapValue(exp, "Name")
			if exportName != nil {
				// We found an export. Store the value for later when we go look for Fn::ImportValue
				_, exportVal, _ := s11n.GetMapValue(val, "Value")
				if exportVal != nil {
					exportMap[exportName.Value] = exportVal
				} else {
					config.Debugf("Unexpected: %s does not have an Export Value", name)
				}
			} else {
				config.Debugf("Unexpected: %s does not have an Export Name", name)
			}
		}
	}

	if len(exportMap) > 0 {
		config.Debugf("exportMap: %v", exportMap)

		vf := func(n *visitor.Visitor) {
			yamlNode := n.GetYamlNode()
			if yamlNode.Kind == yaml.MappingNode {
				if len(yamlNode.Content) == 2 && yamlNode.Content[0].Value == "Fn::ImportValue" {
					exportVal, ok := exportMap[yamlNode.Content[1].Value]
					if ok {
						// We found a match
						config.Debugf("Replacing %s with %s", yamlNode.Content[1].Value, node.ToSJson(exportVal))
						*yamlNode = *node.Clone(exportVal)
					}
				}
			}

		}

		visitor := visitor.NewVisitor(t.Node)
		visitor.Visit(vf)
	}

	return t, nil
}
