package pkg

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/cft/visitor"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// rainConstant parses a !Rain::Constant node
// Constants can be any type of YAML node, but if the constant is a string, it
// can be used in a Sub with ${Rain::ConstantName}. Otherwise use a directive.
// !Rain::Constant ConstantName. Constants are evaluated in order, so they can
// refer to other constants declared previously.
func rainConstant(ctx *directiveContext) (bool, error) {

	config.Debugf("Found a rain constant: %s", node.ToSJson(ctx.n))
	name := ctx.n.Content[1].Value
	val, ok := ctx.t.Constants[name]
	if !ok {
		return false, fmt.Errorf("rain constant %s not found", name)
	}

	*ctx.n = *val

	return true, nil
}

// replaceConstants replaces ${Rain::ConstantName} and ${Const::} in a single
// scalar node If the constant name is not found in the map created from the
// Rain section In the template, an error is returned
func replaceConstants(n *yaml.Node, constants map[string]*yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("expected n to be a ScalarNode")
	}

	// Parse every scalar as if it was a Sub. Look for ${Rain::X}

	retval := ""
	words, err := parse.ParseSub(n.Value, true)
	if err != nil {
		return err
	}
	for _, w := range words {
		switch w.T {
		case parse.STR:
			retval += w.W
		case parse.REF:
			retval += fmt.Sprintf("${%s}", w.W)
		case parse.AWS:
			retval += fmt.Sprintf("${AWS::%s}", w.W)
		case parse.GETATT:
			retval += fmt.Sprintf("${%s}", w.W)
		case parse.RAIN:
			val, ok := constants[w.W]
			if !ok {
				return fmt.Errorf("did not find Rain constant %s", w.W)
			}
			retval += val.Value
		}
	}

	config.Debugf("Replacing %s with %s", n.Value, retval)
	n.Value = retval

	return nil
}

// replaceTemplateConstants scans the entire template looking for Sub strings
// and replaces all instances of ${Rain::ConstantName} if that name exists
// in the Rain/Constants section of the template
func replaceTemplateConstants(templateNode *yaml.Node, constants map[string]*yaml.Node) {

	config.Debugf("Constants: %v", constants)

	vf := func(n *visitor.Visitor) {
		yamlNode := n.GetYamlNode()
		if yamlNode.Kind == yaml.MappingNode {
			if len(yamlNode.Content) == 2 && yamlNode.Content[0].Value == "Fn::Sub" {
				config.Debugf("About to replace constants in %s", yamlNode.Content[1].Value)
				err := replaceConstants(yamlNode.Content[1], constants)
				if err != nil {
					config.Debugf("%v", err)
				}

				// Remove unnecessary Subs
				// Parse the value again and see if it has any non-words
				if !parse.IsSubNeeded(yamlNode.Content[1].Value) {
					config.Debugf("Sub is not needed for %s", yamlNode.Content[1].Value)
					*yamlNode = yaml.Node{Kind: yaml.ScalarNode, Value: yamlNode.Content[1].Value}
				}

			}
		}
	}

	visitor := visitor.NewVisitor(templateNode)
	visitor.Visit(vf)
}

func processConstants(t *cft.Template, n *yaml.Node) error {
	// Process constants in order, since they can refer to previous ones
	_, c, _ := s11n.GetMapValue(n, "Constants")
	if c != nil {
		for i := 0; i < len(c.Content); i += 2 {
			name := c.Content[i].Value
			val := c.Content[i+1]
			t.Constants[name] = val
			// Visit each node in val looking for prior constant entries
			vf := func(v *visitor.Visitor) {
				vnode := v.GetYamlNode()
				if vnode.Kind == yaml.ScalarNode {
					err := replaceConstants(vnode, t.Constants)
					if err != nil {
						// These constants must be scalars
						// TODO: Constant values can be objects!
						config.Debugf("replaceConstants failed: %v", err)
					}
				}
			}
			v := visitor.NewVisitor(val)
			v.Visit(vf)
		}
	}
	return nil
}
