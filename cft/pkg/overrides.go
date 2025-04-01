package pkg

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// validateOverrides returns an error if one of the Overrides isn't found
func (module *Module) ValidateOverrides() error {

	resources := module.ResourcesNode
	moduleParams := module.ParametersNode
	overrides := module.Config.OverridesNode

	config.Debugf("ValidateOverrides resources:\n\n%s\n\nOverrides:\n\n%s\n",
		node.YamlStr(resources), node.YamlStr(overrides))

	// Validate that the overrides actually exist and error if not
	if overrides != nil {
		for i, override := range overrides.Content {
			if i%2 != 0 {
				continue
			}
			foundName := false
			for i, resource := range resources.Content {
				if resource.Kind != yaml.MappingNode {
					continue
				}
				name := resources.Content[i-1].Value
				if name == override.Value {
					foundName = true
					break
				}
			}
			if !foundName {
				return fmt.Errorf("%s override not found: %s",
					module.Config.Name, override.Value)
			}

			// Make sure this Override name is not a module parameter.
			// It is an error to try to override a property that shares
			// a name with a module Parameter.
			if moduleParams != nil {
				_, overrideProps, _ := s11n.GetMapValue(overrides.Content[i+1], Properties)
				if overrideProps != nil {
					for op, overrideProp := range overrideProps.Content {
						if op%2 != 0 {
							continue
						}
						_, mp, _ := s11n.GetMapValue(moduleParams, overrideProp.Value)
						if mp != nil {
							return fmt.Errorf("cannot override module parameter %s",
								overrideProp.Value)
						}
					}
				}
			}
		}
	}
	return nil
}

// processOverrides copies module properties to the new node
// and checks to see if the template overrides anything
func (module *Module) ProcessOverrides(
	resourceName string,
	resource *yaml.Node,
	clonedResource *yaml.Node) error {

	logicalId := module.Config.Name
	moduleConfig := module.Config
	moduleParams := module.ParametersNode

	// Get the overrides from the module config if there are any
	overrides := moduleConfig.ResourceOverridesNode(resourceName)

	// Clone attributes that are like Properties, and replace overridden values
	propLike := []string{Properties, CreationPolicy, Metadata, UpdatePolicy}
	for _, pl := range propLike {
		_, plProps, _ := s11n.GetMapValue(resource, pl)
		_, plOverrides, _ := s11n.GetMapValue(overrides, pl)
		clonedProps := cloneAndReplaceProps(plProps, plOverrides, moduleParams)
		if clonedProps == nil {
			// Was not present in the module or in the template, so skip it
			continue
		}
		if plProps != nil {
			// Get rid of what we cloned, so we can replace it entirely
			node.RemoveFromMap(clonedResource, pl)
		}
		clonedResource.Content = append(clonedResource.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: pl})
		clonedResource.Content = append(clonedResource.Content, clonedProps)
	}

	// DeletionPolicy
	addScalarAttribute(clonedResource, DeletionPolicy, resource, overrides)

	// UpdateReplacePolicy
	addScalarAttribute(clonedResource, UpdateReplacePolicy, resource, overrides)

	// Condition
	addScalarAttribute(clonedResource, Condition, resource, overrides)

	// DependsOn is an array of scalars or a single scalar
	_, moduleDependsOn, _ := s11n.GetMapValue(resource, DependsOn)
	_, templateDependsOn, _ := s11n.GetMapValue(overrides, DependsOn)
	if moduleDependsOn != nil || templateDependsOn != nil {
		clonedResource.Content = append(clonedResource.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: DependsOn})
		dependsOnValue := &yaml.Node{Kind: yaml.SequenceNode, Content: make([]*yaml.Node, 0)}
		if moduleDependsOn != nil {
			// Remove the original DependsOn, so we don't end up with two
			node.RemoveFromMap(clonedResource, DependsOn)

			// Change the names to the modified resource name for the template
			c := make([]*yaml.Node, 0)
			if moduleDependsOn.Kind == yaml.ScalarNode {
				for _, v := range strings.Split(moduleDependsOn.Value, " ") {
					c = append(c, &yaml.Node{Kind: yaml.ScalarNode, Value: v})
				}
			} else {
				// Arrays get converted to space delimited strings
				for _, content := range moduleDependsOn.Content {
					for _, v := range strings.Split(content.Value, " ") {
						c = append(c, &yaml.Node{Kind: yaml.ScalarNode, Value: v})
					}
				}
			}
			for _, r := range c {
				dependsOnValue.Content = append(dependsOnValue.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: rename(logicalId, r.Value)})
			}
		}
		if templateDependsOn != nil {
			if templateDependsOn.Kind == yaml.ScalarNode {
				dependsOnValue.Content = append(dependsOnValue.Content, node.Clone(templateDependsOn))
			} else {
				for _, r := range templateDependsOn.Content {
					dependsOnValue.Content = append(dependsOnValue.Content, node.Clone(r))
				}
			}
		}
		if len(dependsOnValue.Content) == 1 {
			dependsOnValue = node.MakeScalar(dependsOnValue.Content[0].Value)
		}
		clonedResource.Content = append(clonedResource.Content, dependsOnValue)
	}

	return nil
}

// Clone a property-like node from the module and replace any overridden values
func cloneAndReplaceProps(
	moduleProps *yaml.Node,
	overrides *yaml.Node,
	moduleParams *yaml.Node) *yaml.Node {

	// Not all property-like attributes are required
	if moduleProps == nil && overrides == nil {
		return nil
	}

	var props *yaml.Node

	if moduleProps != nil {
		// Start by cloning the properties in the module
		props = node.Clone(moduleProps)
	} else {
		props = &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
	}

	// Replace any property values overridden in the parent template
	if overrides != nil {
		for i, tprop := range overrides.Content {

			// Only look at the names, which have even indexes
			if i%2 != 0 {
				continue
			}

			found := false

			if moduleParams != nil {
				_, moduleParam, _ := s11n.GetMapValue(moduleParams, tprop.Value)

				// Don't clone template props that are module parameters.
				// Module params are used when we resolve Refs later
				if moduleParam != nil {
					continue
				}
			}

			// Overwrite anything hard coded into the module that is
			// present in the parent template
			for j, mprop := range props.Content {
				if tprop.Value == mprop.Value && i%2 == 0 && j%2 == 0 {
					clonedNode := node.Clone(overrides.Content[i+1])
					merged := node.MergeNodes(props.Content[j+1], clonedNode)
					props.Content[j+1] = merged

					found = true
				}
			}

			if !found && i%2 == 0 {
				props.Content = append(props.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: tprop.Value})
				props.Content = append(props.Content, node.Clone(overrides.Content[i+1]))
			}

		}
	}

	return props
}
