package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aws-cloudformation/rain/cft/visitor"
	"github.com/aws-cloudformation/rain/internal/config"
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

// FnMerge merges objects and lists together.
// The arguments must be fully resolvable client-side
func FnMerge(n *yaml.Node) error {

	var err error
	vf := func(v *visitor.Visitor) {
		vn := v.GetYamlNode()
		if vn.Kind != yaml.MappingNode {
			return
		}
		if len(vn.Content) < 2 {
			return
		}
		if vn.Content[0].Value != "Fn::Merge" {
			return
		}
		mrg := vn.Content[1]
		if mrg.Kind != yaml.SequenceNode {
			err = fmt.Errorf("invalid Fn::Merge:\n%s", node.YamlStr(mrg))
			v.Stop()
			return
		}
		if len(mrg.Content) < 2 {
			err = fmt.Errorf("invalid Fn::Merge:\n%s", node.YamlStr(mrg))
			v.Stop()
			return
		}
		merged := &yaml.Node{}
		for _, nodeToMerge := range mrg.Content {
			merged = node.MergeNodes(merged, nodeToMerge)
		}
		*vn = *merged
	}
	visitor.NewVisitor(n).Visit(vf)
	return err
}

// FnSelect reduces Fn::Select to a scalar if it can be fully resolved
func FnSelect(n *yaml.Node) error {

	var err error
	vf := func(v *visitor.Visitor) {
		vn := v.GetYamlNode()
		if vn.Kind != yaml.MappingNode {
			return
		}
		if len(vn.Content) < 2 {
			return
		}
		if vn.Content[0].Value != "Fn::Select" {
			return
		}
		sel := vn.Content[1]
		if len(sel.Content) != 2 {
			err = fmt.Errorf("expected Fn::Select to have 2 elements: %s",
				node.YamlStr(vn))
			v.Stop()
			return
		}
		arr := sel.Content[0]
		if arr.Kind != yaml.SequenceNode {
			return
		}
		idx := sel.Content[1]
		if idx.Kind != yaml.ScalarNode {
			return
		}
		var selected *yaml.Node
		for i, item := range arr.Content {
			if item.Kind != yaml.ScalarNode {
				return
			}
			idxi, converr := strconv.Atoi(idx.Value)
			if converr == nil && i == idxi {
				selected = item
			}
		}
		if selected != nil {
			*vn = *selected
		} else {
			err = fmt.Errorf("invalid Fn::Select, invalid index: %s",
				node.YamlStr(vn))
			v.Stop()
			return
		}
	}
	visitor.NewVisitor(n).Visit(vf)
	return err
}

// FnInsertFile inserts the contents of a local file into the template
func FnInsertFile(n *yaml.Node, basePath string) error {

	config.Debugf("FnInsertFile basePath: %s, n:\n%s", basePath, node.YamlStr(n))
	var err error
	vf := func(v *visitor.Visitor) {
		vn := v.GetYamlNode()
		if vn.Kind != yaml.MappingNode {
			return
		}
		if len(vn.Content) < 2 {
			return
		}
		if vn.Content[0].Value != "Fn::InsertFile" {
			return
		}

		// Get the file path from the node
		filePath := vn.Content[1]
		if filePath.Kind != yaml.ScalarNode {
			err = fmt.Errorf("Fn::InsertFile requires a scalar file path: %s", node.YamlStr(vn))
			v.Stop()
			return
		}

		// Resolve the file path (handle relative paths)
		path := filePath.Value
		if !filepath.IsAbs(path) {
			path = filepath.Join(basePath, path)
		}

		// Read the file contents
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			err = fmt.Errorf("failed to read file %s: %v", path, readErr)
			v.Stop()
			return
		}

		// Replace the node with the file contents as a scalar
		replacement := node.MakeScalar(string(content))
		*vn = *replacement
	}
	visitor.NewVisitor(n).Visit(vf)
	return err
}

// FnInvoke allows treating a module like a function, returning its outputs with modified parameters
func (module *Module) FnInvoke(n *yaml.Node) error {
	var err error
	vf := func(v *visitor.Visitor) {
		vn := v.GetYamlNode()
		if vn.Kind != yaml.MappingNode {
			return
		}
		if len(vn.Content) < 2 {
			return
		}
		if vn.Content[0].Value != "Fn::Invoke" {
			return
		}

		// Get the invoke arguments
		invokeArgs := vn.Content[1]
		if invokeArgs.Kind != yaml.SequenceNode || len(invokeArgs.Content) != 3 {
			err = fmt.Errorf("Fn::Invoke requires 3 arguments [moduleName, parameters, outputs]: %s", node.YamlStr(vn))
			v.Stop()
			return
		}

		// Extract the module name, parameters, and outputs
		moduleName := invokeArgs.Content[0]
		parameters := invokeArgs.Content[1]
		outputs := invokeArgs.Content[2]

		config.Debugf("outputs: %s", node.YamlStr(outputs))

		if moduleName.Kind != yaml.ScalarNode {
			err = fmt.Errorf("Fn::Invoke module name must be a scalar: %s", node.YamlStr(moduleName))
			v.Stop()
			return
		}

		if parameters.Kind != yaml.MappingNode {
			err = fmt.Errorf("Fn::Invoke parameters must be a mapping: %s", node.YamlStr(parameters))
			v.Stop()
			return
		}

		var result *yaml.Node

		// TODO: If the module name does match this module's name, return.
		// TODO: Make a copy of the module and process it
		// TODO: Replace the Fn::Invoke node with the module's outputs

		// Replace the Fn::Invoke node with the result
		*vn = *result
	}
	visitor.NewVisitor(n).Visit(vf)
	return err
}
