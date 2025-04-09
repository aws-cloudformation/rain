package pkg

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// ProcessConditions evaluates conditions in the module and removes Modules and
// Resources that should be omitted by a Condition that evaluates to false. It
// then looks for Fn::If function calls that reference the condition and
// resolves them, removing false nodes.
func (module *Module) ProcessConditions() error {
	// If there are no conditions in the module, nothing to do
	if module.ConditionsNode == nil {
		return nil
	}

	// First resolve condition values
	err := module.Resolve(module.ConditionsNode)
	if err != nil {
		return err
	}

	// Initialize the module's ConditionValues map if it doesn't exist
	if module.ConditionValues == nil {
		module.ConditionValues = make(map[string]bool)
	}

	// Evaluate each condition in the Conditions section in the order they
	// appear in the YAML This ensures that conditions that depend on other
	// conditions are evaluated after their dependencies
	for i := 0; i < len(module.ConditionsNode.Content); i += 2 {
		name := module.ConditionsNode.Content[i].Value
		valNode := module.ConditionsNode.Content[i+1]

		// Evaluate the condition expression directly from the YAML node
		result, err := module.EvalCond(name, valNode)
		if err != nil {
			return err
		}
		module.ConditionValues[name] = result
	}

	sections := []struct {
		name string
		node *yaml.Node
	}{
		{"Resources", module.ResourcesNode},
		{"Modules", module.ModulesNode},
		{"Outputs", module.OutputsNode},
	}

	for _, section := range sections {
		if section.node == nil {
			continue
		}

		// Filter items based on conditions
		// Collect items to remove first to avoid modifying while iterating
		var itemsToRemove []string

		for i := 0; i < len(section.node.Content); i += 2 {
			itemName := section.node.Content[i].Value
			itemNode := section.node.Content[i+1]

			// Check if this item has a Condition attribute
			_, conditionNode, _ := s11n.GetMapValue(itemNode, Condition)
			if conditionNode != nil {
				var conditionResult bool

				if conditionNode.Kind == yaml.ScalarNode {
					conditionName := conditionNode.Value
					conditionResult = module.ConditionValues[conditionName]
				} else {
					return fmt.Errorf("invalid Condition: %s",
						node.YamlStr(itemNode))
				}

				if !conditionResult {
					itemsToRemove = append(itemsToRemove, itemName)
				}

				node.RemoveFromMap(itemNode, Condition)
			}
		}

		// Remove items with false conditions
		for _, itemName := range itemsToRemove {
			err := node.RemoveFromMap(section.node, itemName)
			if err != nil {
				return fmt.Errorf("error removing %s from %s section: %v",
					itemName, section.name, err)
			}
		}

		// Process Fn::If functions in the remaining items
		_, err := module.ProcessFnIf(section.node)
		if err != nil {
			return fmt.Errorf("error processing Fn::If in %s section: %v",
				section.name, err)
		}
	}

	node.RemoveFromMap(module.Node, string(cft.Conditions))

	return nil
}

// EvalCond evaluates a condition expression and returns its boolean value
func (module *Module) EvalCond(name string, val *yaml.Node) (bool, error) {
	config.Debugf("module.EvalCond %s %s:\n%s",
		module.Config.Name, name, node.YamlStr(val))

	// Handle mapping node (most condition functions)
	if val.Kind == yaml.MappingNode && len(val.Content) >= 2 {
		key := val.Content[0].Value
		valueNode := val.Content[1]

		switch key {
		case "Fn::And":
			return module.EvalAnd(valueNode)
		case "Fn::Or":
			return module.EvalOr(valueNode)
		case "Fn::Not":
			return module.EvalNot(valueNode)
		case "Fn::Equals":
			return module.EvalEquals(valueNode)
		case "Condition":
			if valueNode.Kind != yaml.ScalarNode {
				msg := "condition reference must be a string: %s"
				return false, fmt.Errorf(msg, node.YamlStr(valueNode))
			}
			conditionName := valueNode.Value
			// Check if we've already evaluated this condition
			if res, exists := module.ConditionValues[conditionName]; exists {
				return res, nil
			}
			msg := "referenced condition '%s' not found"
			return false, fmt.Errorf(msg, conditionName)
		}
	} else if val.Kind == yaml.ScalarNode {
		// This might be a direct condition reference
		if result, exists := module.ConditionValues[val.Value]; exists {
			return result, nil
		}
	}

	// Default to false if we can't evaluate the condition
	msg := "unable to evaluate condition '%s' in %s: unsupported format: %s"
	return false, fmt.Errorf(msg, name, module.Config.Name, node.YamlStr(val))
}

// EvalAnd evaluates an Fn::And condition
func (module *Module) EvalAnd(n *yaml.Node) (bool, error) {
	if n.Kind != yaml.SequenceNode {
		return false, fmt.Errorf("Fn::And requires a list of conditions: %s", node.YamlStr(n))
	}

	// All conditions must be true for And to be true
	for _, val := range n.Content {
		result, err := module.EvalCond("", val)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil // Short circuit on first false condition
		}
	}
	return true, nil
}

// EvalOr evaluates an Fn::Or condition
func (module *Module) EvalOr(n *yaml.Node) (bool, error) {
	if n.Kind != yaml.SequenceNode {
		msg := "Fn::Or requires a list of conditions: %s"
		return false, fmt.Errorf(msg, node.YamlStr(n))
	}

	// Any condition being true makes Or true
	for _, val := range n.Content {
		result, err := module.EvalCond("", val)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil // Short circuit on first true condition
		}
	}
	return false, nil
}

// EvalNot evaluates an Fn::Not condition
func (module *Module) EvalNot(n *yaml.Node) (bool, error) {
	if n.Kind != yaml.SequenceNode || len(n.Content) != 1 {
		msg := "Fn::Not requires exactly one condition: %s"
		return false, fmt.Errorf(msg, node.YamlStr(n))
	}

	result, err := module.EvalCond("", n.Content[0])
	if err != nil {
		return false, err
	}
	return !result, nil
}

// EvalEquals evaluates an Fn::Equals condition
func (module *Module) EvalEquals(n *yaml.Node) (bool, error) {
	if n.Kind != yaml.SequenceNode || len(n.Content) != 2 {
		msg := "Fn::Equals requires exactly two values: %s"
		return false, fmt.Errorf(msg, node.YamlStr(n))
	}

	// Compare the values - we need to convert to strings for comparison
	val1Str := node.YamlStr(n.Content[0])
	val2Str := node.YamlStr(n.Content[1])

	return val1Str == val2Str, nil
}

// ProcessFnIf processes Fn::If functions in a node and its children
// Returns true if the node should be removed from its parent
func (module *Module) ProcessFnIf(n *yaml.Node) (bool, error) {
	if n == nil {
		return false, nil
	}

	switch n.Kind {
	case yaml.MappingNode:
		// Check if this is an Fn::If node directly
		if len(n.Content) >= 2 && n.Content[0].Value == "Fn::If" {
			value := n.Content[1]
			if value.Kind == yaml.SequenceNode && len(value.Content) == 3 {
				name := value.Content[0].Value
				if condVal, exists := module.ConditionValues[name]; exists {
					// Get the appropriate value based on condition
					var replacement *yaml.Node
					if condVal {
						replacement = node.Clone(value.Content[1])
					} else {
						replacement = node.Clone(value.Content[2])
					}

					// Check if this is AWS::NoValue
					if replacement.Kind == yaml.MappingNode &&
						len(replacement.Content) >= 2 {
						if replacement.Content[0].Value == "Ref" &&
							replacement.Content[1].Value == "AWS::NoValue" {
							return true, nil
						}
					}

					// Replace the entire node with the replacement
					*n = *replacement
					return false, nil
				}
			}
		}

		// Process each key-value pair in the mapping
		i := 0
		for i < len(n.Content) {
			// Make sure we have both key and value
			if i+1 >= len(n.Content) {
				break
			}

			value := n.Content[i+1]

			// Process the value recursively
			shouldRemove, err := module.ProcessFnIf(value)
			if err != nil {
				return false, err
			}

			if shouldRemove {
				// Remove this key-value pair
				if i+2 <= len(n.Content) {
					n.Content = append(n.Content[:i], n.Content[i+2:]...)
				} else {
					n.Content = n.Content[:i]
				}
				// Don't increment i since we've removed elements
				continue
			}

			i += 2
		}

	case yaml.SequenceNode:
		// Process each item in the sequence
		i := 0
		for i < len(n.Content) {
			shouldRemove, err := module.ProcessFnIf(n.Content[i])
			if err != nil {
				return false, err
			}

			if shouldRemove {
				// Remove this item from the sequence
				if i+1 <= len(n.Content) {
					n.Content = append(n.Content[:i], n.Content[i+1:]...)
				} else {
					n.Content = n.Content[:i]
				}
				// Don't increment i since we've removed an element
				continue
			}

			i++
		}
	}

	return false, nil
}
