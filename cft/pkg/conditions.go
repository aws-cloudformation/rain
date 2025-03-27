package pkg

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// ProcessConditions evaluates conditions in the module and
// removes Modules and Resources that should be omitted by
// a Condition that evaluates to false. It then looks for
// Fn::If function calls that reference the condition and
// resolves them, removing the false item.
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

	config.Debugf("ProcessConditions: \n%s", node.YamlStr(module.ConditionsNode))

	// Create a dictionary of condition names to boolean values
	conditionValues := make(map[string]bool)

	// Evaluate each condition in the Conditions section
	conditions := module.Conditions()
	for condName, condValue := range conditions {
		// Evaluate the condition expression
		result, err := evaluateCondition(condName, condValue, conditions, module)
		if err != nil {
			return err
		}
		conditionValues[condName] = result
	}

	config.Debugf("Conditions for %s: %v", module.Config.Name, conditionValues)

	// Process both Resources and Modules sections
	sections := []struct {
		name string
		node *yaml.Node
	}{
		{"Resources", module.ResourcesNode},
		{"Modules", module.ModulesNode},
	}

	numResources := 0
	numModules := 0
	if module.ResourcesNode != nil {
		numResources = len(module.ResourcesNode.Content)
	}
	if module.ModulesNode != nil {
		numModules = len(module.ModulesNode.Content)
	}
	config.Debugf("Module %s has %d Resources and %d Modules",
		module.Config.Name, numResources, numModules)

	for _, section := range sections {
		if section.node == nil {
			continue
		}

		// Filter items based on conditions
		// We need to collect items to remove first to avoid modifying while iterating
		var itemsToRemove []string

		for i := 0; i < len(section.node.Content); i += 2 {
			itemName := section.node.Content[i].Value
			itemNode := section.node.Content[i+1]

			config.Debugf("Checking %s %s", section.name, itemName)

			// Check if this item has a Condition attribute
			_, conditionNode, _ := s11n.GetMapValue(itemNode, Condition)
			if conditionNode != nil {
				var conditionResult bool

				if conditionNode.Kind == yaml.ScalarNode {
					conditionName := conditionNode.Value
					conditionResult = conditionValues[conditionName]
				} else {
					return fmt.Errorf("invalid Condition: %s", node.YamlStr(itemNode))
				}

				if !conditionResult {
					config.Debugf("Removing %s %s", section.name, itemName)

					itemsToRemove = append(itemsToRemove, itemName)
				}

				node.RemoveFromMap(itemNode, Condition)
			}
		}

		// Remove items with false conditions
		for _, itemName := range itemsToRemove {
			err := node.RemoveFromMap(section.node, itemName)
			if err != nil {
				return fmt.Errorf("error removing %s from %s section: %v", itemName, section.name, err)
			}
		}

		// Process Fn::If functions in the remaining items
		_, err := processFnIf(section.node, conditionValues)
		if err != nil {
			return fmt.Errorf("error processing Fn::If in %s section: %v", section.name, err)
		}
	}

	node.RemoveFromMap(module.Node, string(cft.Conditions))

	return nil
}

// evaluateCondition evaluates a CloudFormation condition expression and returns its boolean value
func evaluateCondition(condName string, condValue interface{}, conditions map[string]interface{}, module *Module) (bool, error) {
	// Handle condition node based on its type
	switch v := condValue.(type) {
	case map[string]interface{}:
		// Check for condition functions: Fn::And, Fn::Or, Fn::Not, Fn::Equals, etc.
		if and, ok := v["Fn::And"]; ok {
			return evaluateAnd(and, conditions, module)
		}
		if or, ok := v["Fn::Or"]; ok {
			return evaluateOr(or, conditions, module)
		}
		if not, ok := v["Fn::Not"]; ok {
			return evaluateNot(not, conditions, module)
		}
		if equals, ok := v["Fn::Equals"]; ok {
			return evaluateEquals(equals, module)
		}
		if condition, ok := v["Condition"]; ok {
			// Reference to another condition
			conditionName, ok := condition.(string)
			if !ok {
				return false, fmt.Errorf("condition reference must be a string: %v", condition)
			}
			// Check if we've already evaluated this condition
			if result, exists := conditions[conditionName]; exists {
				boolResult, ok := result.(bool)
				if ok {
					return boolResult, nil
				}
				// If not already evaluated as boolean, recursively evaluate it
				return evaluateCondition(conditionName, result, conditions, module)
			}
			return false, fmt.Errorf("referenced condition '%s' not found", conditionName)
		}
	case string:
		// This might be a direct condition reference
		if result, exists := conditions[v]; exists {
			return evaluateCondition("", result, conditions, module)
		}
	}

	// Default to false if we can't evaluate the condition
	return false, fmt.Errorf("unable to evaluate condition '%s': unsupported format", condName)
}

// evaluateAnd evaluates an Fn::And condition
func evaluateAnd(andExpr interface{}, conditions map[string]interface{}, module *Module) (bool, error) {
	andList, ok := andExpr.([]interface{})
	if !ok {
		return false, fmt.Errorf("Fn::And requires a list of conditions")
	}

	// All conditions must be true for And to be true
	for _, cond := range andList {
		result, err := evaluateCondition("", cond, conditions, module)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil // Short circuit on first false condition
		}
	}
	return true, nil
}

// evaluateOr evaluates an Fn::Or condition
func evaluateOr(orExpr interface{}, conditions map[string]interface{}, module *Module) (bool, error) {
	orList, ok := orExpr.([]interface{})
	if !ok {
		return false, fmt.Errorf("Fn::Or requires a list of conditions")
	}

	// Any condition being true makes Or true
	for _, cond := range orList {
		result, err := evaluateCondition("", cond, conditions, module)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil // Short circuit on first true condition
		}
	}
	return false, nil
}

// evaluateNot evaluates an Fn::Not condition
func evaluateNot(notExpr interface{}, conditions map[string]interface{}, module *Module) (bool, error) {
	notList, ok := notExpr.([]interface{})
	if !ok || len(notList) != 1 {
		return false, fmt.Errorf("Fn::Not requires exactly one condition")
	}

	result, err := evaluateCondition("", notList[0], conditions, module)
	if err != nil {
		return false, err
	}
	return !result, nil
}

// evaluateEquals evaluates an Fn::Equals condition
func evaluateEquals(equalsExpr interface{}, module *Module) (bool, error) {
	equalsList, ok := equalsExpr.([]interface{})
	if !ok || len(equalsList) != 2 {
		return false, fmt.Errorf("Fn::Equals requires exactly two values")
	}

	// Resolve parameter references if needed
	val1 := equalsList[0]
	val2 := equalsList[1]

	// Compare the values
	return val1 == val2, nil
}

// processFnIf processes Fn::If functions in a node and its children
// Returns true if the node should be removed from its parent
func processFnIf(n *yaml.Node, conditionValues map[string]bool) (bool, error) {
	if n == nil {
		return false, nil
	}

	switch n.Kind {
	case yaml.MappingNode:
		// Check if this is an Fn::If node directly
		if len(n.Content) >= 2 && n.Content[0].Value == "Fn::If" {
			value := n.Content[1]
			if value.Kind == yaml.SequenceNode && len(value.Content) == 3 {
				condName := value.Content[0].Value
				if condVal, exists := conditionValues[condName]; exists {
					// Get the appropriate value based on condition
					var replacement *yaml.Node
					if condVal {
						replacement = node.Clone(value.Content[1]) // true value
					} else {
						replacement = node.Clone(value.Content[2]) // false value
					}

					// Check if this is AWS::NoValue
					if replacement.Kind == yaml.MappingNode && len(replacement.Content) >= 2 {
						if replacement.Content[0].Value == "Ref" &&
							replacement.Content[1].Value == "AWS::NoValue" {
							return true, nil // Signal that this node should be removed
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
			shouldRemove, err := processFnIf(value, conditionValues)
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
			shouldRemove, err := processFnIf(n.Content[i], conditionValues)
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
