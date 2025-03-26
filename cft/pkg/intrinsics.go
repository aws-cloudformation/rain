package pkg

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cft/visitor"
	"github.com/aws-cloudformation/rain/internal/node"
	"gopkg.in/yaml.v3"
)

// FnJoin converts to a scalar if the join can be fully resolved
func FnJoin(n *yaml.Node) error {
	var err error
	vf := func(v *visitor.Visitor) {
		vn := v.GetYamlNode()
		if vn.Kind != yaml.MappingNode {
			return
		}
		if len(vn.Content) < 2 {
			return
		}
		if vn.Content[0].Value != "Fn::Join" {
			return
		}
		seq := vn.Content[1]
		if seq.Kind != yaml.SequenceNode {
			err = fmt.Errorf("should be a Sequence: %s", node.YamlStr(vn))
			v.Stop()
			return
		}
		if len(seq.Content) < 2 {
			err = fmt.Errorf("should have length > 1: %s", node.YamlStr(vn))
			v.Stop()
			return
		}

		// Make sure everything is already fully resolved.
		// We don't have to Resolve here since that should have
		// been done already. Just check for Scalars.

		if seq.Content[0].Kind != yaml.ScalarNode {
			return
		}
		separator := seq.Content[0].Value
		items := seq.Content[1]
		if items.Kind != yaml.SequenceNode {
			return
		}
		for _, item := range items.Content {
			if item.Kind != yaml.ScalarNode {
				return
			}
		}
		ss := node.SequenceToStrings(items)
		replacement := node.MakeScalar(strings.Join(ss, separator))
		*vn = *replacement

	}
	visitor.NewVisitor(n).Visit(vf)
	return err
}
