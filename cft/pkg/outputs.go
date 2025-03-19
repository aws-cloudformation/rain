package pkg

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/cft/visitor"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"gopkg.in/yaml.v3"
)

// processModuleOutputs looks for any references in the parent
// template to the module's outputs and replaces them.
func (module *Module) ProcessOutputs() error {

	// Visit each node in the parent template. If we see a Ref, Sub, or
	// GetAtt that points to one of this module's output values,
	// Replace the reference with that value.

	if module == nil {
		return fmt.Errorf("module is nil")
	}

	if module.Config == nil {
		return fmt.Errorf("module config is nil")
	}

	if module.Outputs == nil {
		config.Debugf("module %s has no outputs", module.Config.Name)
		return nil
	}

	// Iterate over module outputs
	for outputName, outputVal := range module.Outputs {
		config.Debugf("processing output %s: %v", outputName, outputVal)

		var err error

		// Use a visitor function to find Fn::Sub or Fn::GetAtt that points to this output
		// Refs can't point to module outputs since you need ModuleName.OutputName
		vf := func(v *visitor.Visitor) {
			n := v.GetYamlNode()
			if n.Kind != yaml.MappingNode || len(n.Content) != 2 {
				return
			}
			switch n.Content[0].Value {
			case string(cft.Sub):
				err = module.OutputSub(outputName, outputVal, n)
				v.Stop()
			case string(cft.GetAtt):
				err = module.OutputGetAtt(outputName, outputVal, n)
				v.Stop()
			default:
				return
			}
		}
		visitor.NewVisitor(module.Parent.Node).Visit(vf)
		if err != nil {
			return err
		}
	}

	return nil
}

// CheckOutputGetAtt checks to see if a GetAtt string matches an Output.
// Returns nil if it's not a match.
func (module *Module) CheckOutputGetAtt(s string, outputName string, outputVal any) (*yaml.Node, error) {
	tokens := strings.Split(s, ".")
	config.Debugf("getOutputAtt %s.%s == %s?", module.Config.Name, outputName, s)
	if len(tokens) != 2 {
		return nil, nil
	}
	if tokens[0] != module.Config.Name {
		return nil, nil
		// TODO: Content[].Arn and Content[0].Arn
	}
	if tokens[1] != outputName {
		return nil, nil
	}
	outputValue, err := encodeOutputValue(outputName, outputVal)
	if err != nil {
		return nil, err
	}
	config.Debugf("getOutputAtt %s.%s returning %v", module.Config.Name, outputName,
		node.ToSJson(outputValue))
	return outputValue, nil
}

// Convert an output value back to a Node.
// Earlier, we converted nodes to maps to make them a little easier to use.
// This also has the benefit of doing a deep copy to avoid
// accidentally referring to the same object repeatedly.
func encodeOutputValue(outputName string, outputVal any) (*yaml.Node, error) {
	var outputNode yaml.Node
	outputValMap, ok := outputVal.(map[string]any)
	// TODO - It could be a plain string.. though this would be rare
	// Output:
	//   Value: foo
	if !ok {
		return nil, fmt.Errorf("output value %s is not a map", outputName)
	}
	val, ok := outputValMap["Value"]
	if !ok {
		return nil, fmt.Errorf("output value %s does not have a Value", outputName)
	}
	err := outputNode.Encode(val)
	if err != nil {
		return nil, err
	}
	return &outputNode, nil
}

// A GetAtt to a module output.
// For example, !GetAtt A.B, where A is a module name, and B is a module output.
// B could be anything, a Scalar or an Object.
func (module *Module) OutputGetAtt(outputName string, outputVal any, n *yaml.Node) error {
	if n.Content[1].Kind != yaml.SequenceNode {
		return fmt.Errorf("expected GetAtt in %s to be a sequence: %s",
			module.Config.Name, node.ToSJson(n))
	}
	ss := sequenceToStrings(n.Content[1])
	o, err := module.CheckOutputGetAtt(strings.Join(ss, "."), outputName, outputVal)
	if err != nil {
		return err
	}
	if o != nil {
		config.Debugf("getatt replacing\n%s\n\nwith\n\n%s", node.ToSJson(n), node.ToSJson(o))
		*n = *o
	}
	return nil
}

// OutputSub checks a Sub to see if it refers to a module Output.
// A Sub string can refer to an output scalar value.
// The reference needs to be like a GetAtt.
// For example, !Sub ${A.B} refers to module A, output B.
func (module *Module) OutputSub(outputName string, outputVal any, n *yaml.Node) error {
	s := n.Content[1].Value
	words, err := parse.ParseSub(s, true)
	if err != nil {
		return err
	}
	sub := ""
	for _, word := range words {
		switch word.T {
		case parse.STR:
			sub += word.W
		case parse.AWS:
			sub += "${AWS::" + word.W + "}"
		case parse.REF:
			sub += "${" + word.W + "}"
		case parse.GETATT:
			config.Debugf("sub GetAtt: %s", word.W)
			outputValue, err := module.CheckOutputGetAtt(word.W, outputName, outputVal)
			if err != nil {
				return err
			}
			if outputValue == nil {
				sub += "${" + word.W + "}"
			} else {
				if outputValue.Kind == yaml.MappingNode {
					// Prepend the module name
					v := outputValue.Content[1]
					switch outputValue.Content[0].Value {
					case string(cft.Sub):
						sub += "${" + module.Config.Name + v.Value + "}"
					case string(cft.GetAtt):
						ss := sequenceToStrings(v)
						joined := strings.Join(ss, ".")
						sub += "${" + module.Config.Name + joined + "}"
					case string(cft.Ref):
						sub += "${" + module.Config.Name + v.Value + "}"
					}
				} else if outputValue.Kind == yaml.ScalarNode {
					sub += outputValue.Value
				}
			}
		}
	}

	var subNode *yaml.Node
	if parse.IsSubNeeded(sub) {
		subNode = node.MakeMappingNode()
		subNode.Content = append(subNode.Content, node.MakeScalarNode(string(cft.Sub)))
		subNode.Content = append(subNode.Content, node.MakeScalarNode(sub))
	} else {
		subNode = node.MakeScalarNode(sub)
	}
	if sub != s {
		config.Debugf("sub replacing\n%s\n\nwith\n\n%s", node.ToSJson(n), node.ToSJson(subNode))
		*n = *subNode
	}
	return nil
}

// sequenceToStrings converts a sequence of Scalars to a string slice
func sequenceToStrings(n *yaml.Node) []string {
	ss := []string{}
	for _, v := range n.Content {
		ss = append(ss, v.Value)
	}
	return ss
}
