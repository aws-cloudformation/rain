package pkg

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// validateOverrides returns an error if one of the Overrides isn't found
func validateOverrides(
	templateResource *yaml.Node,
	moduleResources *yaml.Node,
	moduleParams *yaml.Node) error {

	_, overrides, _ := s11n.GetMapValue(templateResource, Overrides)

	// Validate that the overrides actually exist and error if not
	if overrides != nil {
		for i, override := range overrides.Content {
			if i%2 != 0 {
				continue
			}
			foundName := false
			for i, moduleResource := range moduleResources.Content {
				if moduleResource.Kind != yaml.MappingNode {
					continue
				}
				name := moduleResources.Content[i-1].Value
				if name == override.Value {
					foundName = true
					break
				}
			}
			if !foundName {
				return fmt.Errorf("override not found: %s", override.Value)
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
func processOverrides(
	logicalId string,
	moduleConfig *yaml.Node,
	moduleResourceName string,
	moduleResource *yaml.Node,
	clonedResource *yaml.Node,
	moduleParams *yaml.Node) (*yaml.Node, error) {

	// Get the overrides from the module config if there are any
	var overrides *yaml.Node
	_, allOverrides, _ := s11n.GetMapValue(moduleConfig, Overrides)
	if allOverrides != nil {
		_, overrides, _ = s11n.GetMapValue(allOverrides, moduleResourceName)
	}

	// Clone attributes that are like Properties, and replace overridden values
	propLike := []string{Properties, CreationPolicy, Metadata, UpdatePolicy}
	for _, pl := range propLike {
		_, plProps, _ := s11n.GetMapValue(moduleResource, pl)
		_, plTemplateProps, _ := s11n.GetMapValue(overrides, pl)
		clonedProps := cloneAndReplaceProps(clonedResource, pl, plProps, plTemplateProps, moduleParams)
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
	addScalarAttribute(clonedResource, DeletionPolicy, moduleResource, overrides)

	// UpdateReplacePolicy
	addScalarAttribute(clonedResource, UpdateReplacePolicy, moduleResource, overrides)

	// Condition
	addScalarAttribute(clonedResource, Condition, moduleResource, overrides)

	// DependsOn is an array of scalars or a single scalar
	_, moduleDependsOn, _ := s11n.GetMapValue(moduleResource, DependsOn)
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
		clonedResource.Content = append(clonedResource.Content, dependsOnValue)
	}

	return overrides, nil
}
