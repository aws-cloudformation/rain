// This file implements !Rain::Module
package pkg

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// Clone a property-like node from the module and replace any overridden values
func cloneAndReplaceProps(
	ext *yaml.Node,
	name string,
	moduleProps *yaml.Node,
	templateProps *yaml.Node,
	moduleParams *yaml.Node) *yaml.Node {

	// Not all property-like attributes are required
	if moduleProps == nil {
		return nil
	}

	// Add the node to the output
	ext.Content = append(ext.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: name})

	// Start by cloning the properties in the module
	props := node.Clone(moduleProps)

	// Replace any property values overridden in the parent template
	if templateProps != nil {
		for i, tprop := range templateProps.Content {

			// Only look at the names, which have even indexes
			if i%2 != 0 {
				continue
			}

			found := false

			if moduleParams != nil {
				_, moduleParam := s11n.GetMapValue(moduleParams, tprop.Value)

				// Don't clone template props that are module parameters.
				// Module params are used when we resolve Refs later
				if moduleParam != nil {
					continue
				}
			}

			// Override anything hard coded into the module that is present in the parent template
			for j, mprop := range props.Content {
				// Property names are even-indexed array elements
				if tprop.Value == mprop.Value && i%2 == 0 && j%2 == 0 {
					// Is a clone good enough here? Could get weird.
					// Maybe we just require that you replace the entire property if it's nested
					// Otherwise we have to do a diff
					props.Content[j+1] = node.Clone(templateProps.Content[i+1])
					found = true
				}
			}

			if !found && i%2 == 0 {
				props.Content = append(props.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: tprop.Value})
				props.Content = append(props.Content, node.Clone(templateProps.Content[i+1]))
			}

		}
	}

	ext.Content = append(ext.Content, props)
	return props
}

// Add DeletionPolicy and UpdateReplacePolicy
func addPolicy(ext *yaml.Node, name string, moduleExtension *yaml.Node, templateResource *yaml.Node) {
	_, templatePolicy := s11n.GetMapValue(templateResource, name)
	_, modulePolicy := s11n.GetMapValue(moduleExtension, name)
	if templatePolicy != nil || modulePolicy != nil {
		policy := &yaml.Node{Kind: yaml.ScalarNode, Value: name}
		var policyValue *yaml.Node
		if templatePolicy != nil {
			policyValue = node.Clone(templatePolicy)
		} else {
			policyValue = node.Clone(modulePolicy)
		}
		ext.Content = append(ext.Content, policy)
		ext.Content = append(ext.Content, policyValue)
	}
}

// Rename a resource defined in the module to add the template resource name
func rename(logicalId string, resourceName string) string {
	if resourceName == "ModuleExtension" {
		return logicalId
	} else {
		return logicalId + resourceName
	}
}

// Recursive function to find all refs in properties
// Also handles DeletionPolicy, UpdateRetainPolicy
func renamePropRefs(
	parentName string,
	propName string,
	prop *yaml.Node,
	ext *yaml.Node,
	moduleParams *yaml.Node,
	moduleResources *yaml.Node,
	logicalId string,
	templateProps *yaml.Node) error {

	// Properties:
	//   SimpleProp: Val
	//   RefParam: !Ref NameOfParam
	//   RefResource: !Ref NameOfResource
	//   GetAtt: !GetAtt ModuleExtension.Arn
	//   Complex:
	//     AnArray:
	//       - Element0
	//           A: B
	//           C: !Ref D

	config.Debugf("renamePropRefs parentName: %v, propName: %v, prop.Kind: %v", parentName, propName, prop.Kind)

	if prop.Kind == yaml.ScalarNode {
		refFoundInParams := false
		if propName == "Ref" {
			if moduleParams != nil {
				// Find the module parameter that matches the !Ref
				_, param := s11n.GetMapValue(moduleParams, prop.Value)
				if param != nil {
					// We need to get the parameter value from the parent template.
					// Module params are set by the parent template resource properties.

					// Look for this property name in the parent template
					_, parentVal := s11n.GetMapValue(templateProps, prop.Value)
					if parentVal == nil {
						return fmt.Errorf("did not find %v in parent template Resource Properties", prop.Value)
					}

					config.Debugf("parentVal: %v", node.ToJson(parentVal))

					// We can't just set prop.Value, since we would end up with Prop: !Ref Value instead of just Prop: Value
					// Get the property's parent and set the entire map value for the property

					// Get the map parent within the extension node we created
					refMap := node.GetParent(prop, ext, nil)
					if refMap.Value == nil {
						return fmt.Errorf("could not find parent for %v", prop)
					}
					propParentPair := node.GetParent(refMap.Value, ext, nil)

					config.Debugf("propParentPair.Value: %v", node.ToJson(propParentPair.Value))

					// Create a new node to replace what's defined in the module
					newValue := node.Clone(parentVal)

					config.Debugf("Setting %v to:\n%v", parentName, node.ToJson(newValue))

					node.SetMapValue(propParentPair.Value, parentName, newValue)

					refFoundInParams = true
				}
			}
			if !refFoundInParams {
				config.Debugf("Did not find param for %v", prop.Value)
				// Look for a resource in the module
				_, resource := s11n.GetMapValue(moduleResources, prop.Value)
				if resource == nil {
					config.Debugf("did not find !Ref %v", prop.Value)
					// If we can't find the Ref, leave it alone and assume it's
					// expected to be in the parent template to be resolved at deploy time.
					return nil
				}
				fixedName := rename(logicalId, prop.Value)
				prop.Value = fixedName
			}
		}
	} else if prop.Kind == yaml.SequenceNode {
		config.Debugf("Sequence %v %v", propName, node.ToJson(prop))
		if propName == "Fn::GetAtt" {
			// Convert !GetAtt ModuleExtension.Property to !GetAtt LogicalId.Property
			fixedName := rename(logicalId, prop.Content[0].Value)
			prop.Content[0].Value = fixedName
			config.Debugf("Fixed GetAtt: %v", fixedName)
		} else {
			// Recurse over array elements
			for i, p := range prop.Content {
				result := renamePropRefs(propName,
					p.Value, prop.Content[i], ext, moduleParams, moduleResources, logicalId, templateProps)
				if result != nil {
					return result
				}
			}
		}

	} else if prop.Kind == yaml.MappingNode {
		config.Debugf("Mapping %v %v", propName, node.ToJson(prop))
		for i, p := range prop.Content {
			if i%2 == 0 {
				config.Debugf("About to renamePropRefs for Mapping Content: %v", p.Value)
				result := renamePropRefs(propName,
					p.Value, prop.Content[i+1], ext, moduleParams, moduleResources, logicalId, templateProps)
				if result != nil {
					return result
				}
			}
		}
	} else {
		return fmt.Errorf("unexpected prop Kind: %v", prop.Kind)
	}

	return nil
}

// Convert !Ref values
func resolveRefs(ext *yaml.Node, moduleParams *yaml.Node,
	moduleResources *yaml.Node, logicalId string, templateProps *yaml.Node) error {
	// Replace references to the module's parameters with the value supplied
	// by the parent template. Rename refs to other resources in the module.

	config.Debugf("resolveRefs ext: %v", node.ToJson(ext))

	_, extProps := s11n.GetMapValue(ext, "Properties")
	if extProps != nil {
		for i, prop := range extProps.Content {
			if i%2 == 0 {
				propName := prop.Value
				config.Debugf("Resolving refs for %v", propName)
				err := renamePropRefs(propName,
					propName, extProps.Content[i+1], ext, moduleParams, moduleResources, logicalId, templateProps)
				if err != nil {
					config.Debugf("%v", err)
					return fmt.Errorf("unable to resolve refs for %v", propName)
				}
			}
		}
	}

	// DeletionPolicy, UpdateReplacePolicy
	policies := []string{"DeletionPolicy", "UpdateReplacePolicy"}
	for _, policy := range policies {
		_, policyNode := s11n.GetMapValue(ext, policy)
		if policyNode != nil {
			config.Debugf("policyNode: %v", node.ToJson(policyNode))
			err := renamePropRefs(policy, policy, policyNode, ext, moduleParams, moduleResources, logicalId, templateProps)
			if err != nil {
				config.Debugf("%v", err)
				return fmt.Errorf("unable to resolve refs for %v", policy)
			}
		}
	}

	return nil
}

// Convert the module into a node for the packaged template
func processModule(module *yaml.Node,
	outputNode *yaml.Node, t cft.Template,
	typeNode *yaml.Node, parent node.NodePair) (bool, error) {

	// The parent arg is the map in the template resource's Content[1] that contains Type, Properties, etc

	if parent.Key == nil {
		return false, errors.New("expected parent.Key to not be nil. The !Rain::Module directive should come after Type: ")
	}

	// Get the logical id of the resource we are transforming
	logicalId := parent.Key.Value

	// Make a new node that will hold our additions to the original template
	outputNode.Content = make([]*yaml.Node, 0)

	if module.Kind != yaml.DocumentNode {
		return false, errors.New("expected module to be a DocumentNode")
	}

	curNode := module.Content[0] // ScalarNode !!map

	// Locate the Resources: section in the module
	_, moduleResources := s11n.GetMapValue(curNode, "Resources")

	if moduleResources == nil {
		return false, errors.New("expected the module to have a Resources section")
	}

	// Locate the Parameters: section in the module (might be nil)
	_, moduleParams := s11n.GetMapValue(curNode, "Parameters")

	templateResource := parent.Value // The !!map node of the resource with Type !Rain::Module
	_, templateProps := s11n.GetMapValue(templateResource, "Properties")
	if templateProps == nil {
		return false, errors.New("expected template resource to have Properties")
	}

	// Locate the ModuleExtension: resource. There should be 0 or 1
	_, moduleExtension := s11n.GetMapValue(moduleResources, "ModuleExtension")
	if moduleExtension == nil {
		config.Debugf("the module does not have a ModuleExtension resource")
	} else {

		// Process the ModuleExtension resource.

		// Add the logical id to the output node, which will replace the original
		outputNode.Content = append(outputNode.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: logicalId, // This is the logical name of the resource from the parent template
		})

		_, meta := s11n.GetMapValue(moduleExtension, "Metadata")
		if meta == nil {
			return false, errors.New("expected ModuleExtension.Metadata")
		}

		_, extends := s11n.GetMapValue(meta, "Extends")
		if extends == nil {
			return false, errors.New("expected ModuleExtension.Metadata.Extends")
		}

		// Create a new node to contain the extended resource.
		// This will be added to the template, and the original resource node will be removed
		ext := &yaml.Node{}
		ext.Kind = yaml.MappingNode
		ext.Content = make([]*yaml.Node, 0)

		// Type:
		// Replace the !Rain::Module directive with the extended type from the module
		ext.Content = append(ext.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "Type"})
		ext.Content = append(ext.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: extends.Value})

		// Clone attributes that are like Properties, and replace overridden values
		propLike := []string{"Properties", "CreationPolicy", "Metadata", "UpdatePolicy"}
		for _, pl := range propLike {
			_, plProps := s11n.GetMapValue(moduleExtension, pl)
			_, plTemplateProps := s11n.GetMapValue(templateResource, pl)
			outProps := cloneAndReplaceProps(ext, pl, plProps, plTemplateProps, moduleParams)
			if pl == "Metadata" {
				// Remove the Extends attribute
				node.RemoveFromMap(outProps, "Extends")
			}
		}

		// DeletionPolicy
		addPolicy(ext, "DeletionPolicy", moduleExtension, templateResource)

		// UpdateReplacePolicy
		addPolicy(ext, "UpdateReplacePolicy", moduleExtension, templateResource)

		// DependsOn is an array of scalars or a single scalar
		_, moduleDependsOn := s11n.GetMapValue(moduleExtension, "DependsOn")
		_, templateDependsOn := s11n.GetMapValue(templateResource, "DependsOn")
		if moduleDependsOn != nil || templateDependsOn != nil {
			ext.Content = append(ext.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "DependsOn"})
			dependsOnValue := &yaml.Node{Kind: yaml.SequenceNode, Content: make([]*yaml.Node, 0)}
			if moduleDependsOn != nil {
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
			ext.Content = append(ext.Content, dependsOnValue)
		}

		// Resolve Refs in the module extension
		// Some refs are to other resources in the module
		// Other refs are to the module's parameters
		resolveRefs(ext, moduleParams, moduleResources, logicalId, templateProps)

		// Add the Condition from the parent template
		_, parentCondition := s11n.GetMapValue(templateResource, "Condition")
		if parentCondition != nil {
			ext.Content = append(ext.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "Condition"})
			ext.Content = append(ext.Content, node.Clone(parentCondition))
		}

		// Add the extension to the output node
		outputNode.Content = append(outputNode.Content, ext)
	}

	// Get additional resources and add them to the output
	for i, resource := range moduleResources.Content {
		if resource.Kind == yaml.MappingNode {
			name := moduleResources.Content[i-1].Value
			if name != "ModuleExtension" {

				// Resolve Conditions. Rain handles this differently, since a rain
				// module cannot have a Condition section. This value must be a module parameter
				// name, and the value must be set in the parent template as the name of
				// a Condition that is defined in the parent.
				_, condition := s11n.GetMapValue(resource, "Condition")
				if condition != nil {
					conditionErr := errors.New("a Condition in a rain module must be the name of a Parameter that is set the name of a Condition in the parent template")
					// The value must be present in the module's parameters
					if condition.Kind != yaml.ScalarNode {
						return false, conditionErr
					}
					_, param := s11n.GetMapValue(moduleParams, condition.Value)
					if param == nil {
						return false, conditionErr
					}
					_, conditionVal := s11n.GetMapValue(templateProps, condition.Value)
					if conditionVal == nil {
						return false, conditionErr
					}
					if conditionVal.Kind != yaml.ScalarNode {
						return false, conditionErr
					}
					condition.Value = conditionVal.Value
				}

				// This is an additional resource to be added
				nameNode := node.Clone(moduleResources.Content[i-1])
				nameNode.Value = rename(logicalId, nameNode.Value)
				outputNode.Content = append(outputNode.Content, nameNode)
				clonedResource := node.Clone(resource)
				// Modify referenced resource names
				_, dependsOn := s11n.GetMapValue(clonedResource, "DependsOn")
				if dependsOn != nil {
					replaceDependsOn := &yaml.Node{Kind: yaml.SequenceNode, Content: make([]*yaml.Node, 0)}
					if dependsOn.Kind == yaml.ScalarNode {
						for _, v := range strings.Split(dependsOn.Value, " ") {
							replaceDependsOn.Content = append(replaceDependsOn.Content,
								&yaml.Node{Kind: yaml.ScalarNode, Value: rename(logicalId, v)})
						}
					} else {
						for _, c := range dependsOn.Content {
							for _, v := range strings.Split(c.Value, " ") {
								replaceDependsOn.Content = append(replaceDependsOn.Content,
									&yaml.Node{Kind: yaml.ScalarNode, Value: rename(logicalId, v)})
							}
						}
					}
					node.SetMapValue(clonedResource, "DependsOn", replaceDependsOn)
				}
				resolveRefs(clonedResource, moduleParams, moduleResources, logicalId, templateProps)
				outputNode.Content = append(outputNode.Content, clonedResource)
			}
		}
	}

	return true, nil
}

// Type: !Rain::Module
func module(n *yaml.Node, root string, t cft.Template, parent node.NodePair) (bool, error) {

	if !Experimental {
		panic("You must add the --experimental arg to use the ~Rain:Module directive")
	}

	if len(n.Content) != 2 {
		return false, errors.New("expected !Rain::Module <URI>")
	}

	uri := n.Content[1].Value
	var content []byte
	var err error
	var path string

	// Is this a local file or a URL?
	if strings.HasPrefix(uri, "https://") {
		// Download the file and then parse it
		resp, err := http.Get(uri)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		content = []byte(body)
	} else {
		// Read the local file
		content, path, err = expectFile(n, root)
		if err != nil {
			return false, err
		}
	}

	// Parse the file
	var moduleNode yaml.Node
	err = yaml.Unmarshal(content, &moduleNode)
	if err != nil {
		return false, err
	}

	// Transform
	parse.TransformNode(&moduleNode)
	// TODO: I think this allows us to nest modules. Test it.
	_, err = transform(&moduleNode, filepath.Dir(path), t)
	if err != nil {
		return false, err
	}

	// Create a new node to represent the processed module
	var outputNode yaml.Node
	_, err = processModule(&moduleNode, &outputNode, t, n, parent)
	if err != nil {
		return false, err
	}

	// Find the resource node in the template
	_, resourceNode := s11n.GetMapValue(t.Node.Content[0], "Resources")
	if resourceNode == nil {
		return false, errors.New("expected template to have Resources")
	}

	// j, _ := json.MarshalIndent(resourceNode, "", "  ")
	// config.Debugf("resourceNode: %v", string(j))

	// j, _ = json.MarshalIndent(outputNode, "", "  ")
	// config.Debugf("outputNode: %v", string(j))

	// Remove the original from the template
	err = node.RemoveFromMap(resourceNode, parent.Key.Value)
	if err != nil {
		return false, err
	}

	// Insert the transformed resource into the template
	resourceNode.Content = append(resourceNode.Content, outputNode.Content...)

	return true, nil

}
