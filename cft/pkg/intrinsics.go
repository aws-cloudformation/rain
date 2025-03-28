package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/visitor"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
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
			err = fmt.Errorf("Fn::Invoke requires 3 arguments [moduleName, parameters, outputKey]: %s", node.YamlStr(vn))
			v.Stop()
			return
		}

		// Extract the module name, parameters, and output key
		moduleName := invokeArgs.Content[0]
		parameters := invokeArgs.Content[1]
		outputKey := invokeArgs.Content[2]

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

		if outputKey.Kind != yaml.ScalarNode {
			err = fmt.Errorf("Fn::Invoke output key must be a scalar: %s", node.YamlStr(outputKey))
			v.Stop()
			return
		}

		// Find the module in the parent template's Modules section
		moduleNameStr := moduleName.Value

		// Check if this module name matches the current module
		if module.Config != nil && module.Config.Name == moduleNameStr {
			err = fmt.Errorf("cannot invoke the current module (%s) from within itself", moduleNameStr)
			v.Stop()
			return
		}

		// Find the module in the parent template
		var moduleConfig *cft.ModuleConfig
		var moduleNode *yaml.Node

		if module.Parent != nil && module.Parent.Node != nil {
			// Look for the module in the Modules section
			_, modulesSection, _ := s11n.GetMapValue(module.Parent.Node.Content[0], "Modules")
			if modulesSection != nil && modulesSection.Kind == yaml.MappingNode {
				for i := 0; i < len(modulesSection.Content); i += 2 {
					if modulesSection.Content[i].Value == moduleNameStr {
						// Found the module
						moduleNode = modulesSection.Content[i+1]
						var parseErr error
						moduleConfig, parseErr = cft.ParseModuleConfig(moduleNameStr, moduleNode)
						if parseErr != nil {
							err = fmt.Errorf("failed to parse module config for %s: %v", moduleNameStr, parseErr)
							v.Stop()
							return
						}
						break
					}
				}
			}
		}

		if moduleConfig == nil {
			err = fmt.Errorf("module %s not found in parent template", moduleNameStr)
			v.Stop()
			return
		}

		// Get the module content
		moduleContent, getErr := getModuleContent(module.Parsed.RootDir, module.Parent, module.Parsed.FS, "", moduleConfig.Source)
		if getErr != nil {
			err = fmt.Errorf("failed to get module content for %s: %v", moduleNameStr, getErr)
			v.Stop()
			return
		}

		// Parse the module
		parsed, parseErr := parseModule(moduleContent.Content, moduleContent.NewRootDir, module.Parsed.FS)
		if parseErr != nil {
			err = fmt.Errorf("failed to parse module %s: %v", moduleNameStr, parseErr)
			v.Stop()
			return
		}

		// Create a copy of the module config with the overridden parameters
		newModuleConfig := *moduleConfig

		// Create a deep copy of the PropertiesNode or create a new one if it doesn't exist
		if moduleConfig.PropertiesNode != nil {
			newModuleConfig.PropertiesNode = node.Clone(moduleConfig.PropertiesNode)
		} else {
			newModuleConfig.PropertiesNode = node.MakeMapping()
		}

		// Merge the parameters from the Fn::Invoke into the module config
		for i := 0; i < len(parameters.Content); i += 2 {
			paramName := parameters.Content[i].Value
			paramValue := parameters.Content[i+1]

			// Add or replace the parameter in the module config
			node.RemoveFromMap(newModuleConfig.PropertiesNode, paramName)
			newModuleConfig.PropertiesNode.Content = append(newModuleConfig.PropertiesNode.Content,
				node.Clone(parameters.Content[i]), node.Clone(paramValue))
		}

		// Create a temporary module to process
		tempModule := &Module{
			Config: &newModuleConfig,
			Node:   node.Clone(parsed.Node),
			Parent: module.Parent,
			Parsed: parsed,
		}
		tempModule.InitNodes()

		// Process the module to evaluate conditions and resolve references
		err = tempModule.ProcessConditions()
		if err != nil {
			err = fmt.Errorf("failed to process conditions in module %s: %v", moduleNameStr, err)
			v.Stop()
			return
		}

		// Find the output with the specified key
		outputKeyStr := outputKey.Value
		var result *yaml.Node

		// Look for the output in the module's Outputs section
		if tempModule.OutputsNode != nil {
			for i := 0; i < len(tempModule.OutputsNode.Content); i += 2 {
				if tempModule.OutputsNode.Content[i].Value == outputKeyStr {
					// Found the output
					outputObj := tempModule.OutputsNode.Content[i+1]
					_, valueNode, _ := s11n.GetMapValue(outputObj, "Value")
					if valueNode != nil {
						// Clone the output value
						result = node.Clone(valueNode)

						// Resolve references in the output value
						err = tempModule.Resolve(result)
						if err != nil {
							err = fmt.Errorf("failed to resolve references in output %s: %v", outputKeyStr, err)
							v.Stop()
							return
						}

						// Process any intrinsic functions in the result
						err = ExtraIntrinsics(result, tempModule.Parsed.RootDir)
						if err != nil {
							err = fmt.Errorf("failed to process intrinsic functions in output %s: %v", outputKeyStr, err)
							v.Stop()
							return
						}

						break
					}
				}
			}
		}

		if result == nil {
			err = fmt.Errorf("output %s not found in module %s", outputKeyStr, moduleNameStr)
			v.Stop()
			return
		}

		// Replace the Fn::Invoke node with the result
		*vn = *result
	}
	visitor.NewVisitor(n).Visit(vf)
	return err
}
