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
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// Clone a property-like node from the module and replace any overridden values
func cloneAndReplaceProps(
	n *yaml.Node,
	name string,
	moduleProps *yaml.Node,
	templateProps *yaml.Node,
	moduleParams *yaml.Node) *yaml.Node {

	// Not all property-like attributes are required
	if moduleProps == nil && templateProps == nil {
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
	if templateProps != nil {
		for i, tprop := range templateProps.Content {

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

	return props
}

// Add DeletionPolicy and UpdateReplacePolicy
func addPolicy(out *yaml.Node, name string, moduleResource *yaml.Node, templateOverrides *yaml.Node) {
	_, templatePolicy, _ := s11n.GetMapValue(templateOverrides, name)
	_, modulePolicy, _ := s11n.GetMapValue(moduleResource, name)
	if templatePolicy != nil || modulePolicy != nil {
		policy := &yaml.Node{Kind: yaml.ScalarNode, Value: name}
		var policyValue *yaml.Node
		if templatePolicy != nil {
			policyValue = node.Clone(templatePolicy)
		} else {
			policyValue = node.Clone(modulePolicy)
		}
		out.Content = append(out.Content, policy)
		out.Content = append(out.Content, policyValue)
	}
}

// Rename a resource defined in the module to add the template resource name
func rename(logicalId string, resourceName string) string {
	return logicalId + resourceName
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
	//   GetAtt: !GetAtt Name.Arn
	//   Complex:
	//     AnArray:
	//       - Element0
	//           A: B
	//           C: !Ref D

	// config.Debugf("renamePropRefs parentName: %v, propName: %v, prop.Kind: %v", parentName, propName, prop.Kind)

	if prop.Kind == yaml.ScalarNode {
		refFoundInParams := false
		if propName == "Ref" {
			if moduleParams != nil {
				// Find the module parameter that matches the !Ref
				_, param, _ := s11n.GetMapValue(moduleParams, prop.Value)
				if param != nil {
					// We need to get the parameter value from the parent template.
					// Module params are set by the parent template resource properties.

					// Look for this property name in the parent template
					_, parentVal, _ := s11n.GetMapValue(templateProps, prop.Value)
					if parentVal == nil {
						return fmt.Errorf("did not find %v in parent template Resource Properties", prop.Value)
					}

					// config.Debugf("parentVal: %v", node.ToJson(parentVal))

					// We can't just set prop.Value, since we would end up with Prop: !Ref Value instead of just Prop: Value
					// Get the property's parent and set the entire map value for the property

					// Get the map parent within the extension node we created
					refMap := node.GetParent(prop, ext, nil)
					if refMap.Value == nil {
						return fmt.Errorf("could not find parent for %v", prop)
					}
					propParentPair := node.GetParent(refMap.Value, ext, nil)

					// config.Debugf("propParentPair.Value: %v", node.ToJson(propParentPair.Value))

					// Create a new node to replace what's defined in the module
					newValue := node.Clone(parentVal)

					// config.Debugf("Setting %v to:\n%v", parentName, node.ToJson(newValue))

					node.SetMapValue(propParentPair.Value, parentName, newValue)

					refFoundInParams = true
				}
			}
			if !refFoundInParams {
				// config.Debugf("Did not find param for %v", prop.Value)
				// Look for a resource in the module
				_, resource, _ := s11n.GetMapValue(moduleResources, prop.Value)
				if resource == nil {
					// config.Debugf("did not find !Ref %v", prop.Value)
					// If we can't find the Ref, leave it alone and assume it's
					// expected to be in the parent template to be resolved at deploy time.
					return nil
				}
				fixedName := rename(logicalId, prop.Value)
				prop.Value = fixedName
			}
		}
	} else if prop.Kind == yaml.SequenceNode {
		// config.Debugf("Sequence %v %v", propName, node.ToJson(prop))
		if propName == "Fn::GetAtt" {
			// Convert !GetAtt Name.Property to !GetAtt LogicalId.Property
			fixedName := rename(logicalId, prop.Content[0].Value)
			prop.Content[0].Value = fixedName
			// config.Debugf("Fixed GetAtt: %v", fixedName)
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
		// config.Debugf("Mapping %v %v", propName, node.ToJson(prop))
		for i, p := range prop.Content {
			if i%2 == 0 {
				// config.Debugf("About to renamePropRefs for Mapping Content: %v", p.Value)
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
func resolveRefs(
	ext *yaml.Node,
	moduleParams *yaml.Node,
	moduleResources *yaml.Node,
	logicalId string,
	templateProps *yaml.Node) error {

	// Replace references to the module's parameters with the value supplied
	// by the parent template. Rename refs to other resources in the module.

	// config.Debugf("resolveRefs ext: %v", node.ToJson(ext))

	_, extProps, _ := s11n.GetMapValue(ext, "Properties")
	if extProps != nil {
		for i, prop := range extProps.Content {
			if i%2 == 0 {
				propName := prop.Value
				// config.Debugf("Resolving refs for %v", propName)
				err := renamePropRefs(propName,
					propName, extProps.Content[i+1], ext, moduleParams, moduleResources, logicalId, templateProps)
				if err != nil {
					// config.Debugf("%v", err)
					return fmt.Errorf("unable to resolve refs for %v", propName)
				}
			}
		}
	}

	// DeletionPolicy, UpdateReplacePolicy
	policies := []string{"DeletionPolicy", "UpdateReplacePolicy"}
	for _, policy := range policies {
		_, policyNode, _ := s11n.GetMapValue(ext, policy)
		if policyNode != nil {
			// config.Debugf("policyNode: %v", node.ToJson(policyNode))
			err := renamePropRefs(policy, policy, policyNode, ext, moduleParams, moduleResources, logicalId, templateProps)
			if err != nil {
				// config.Debugf("%v", err)
				return fmt.Errorf("unable to resolve refs for %v", policy)
			}
		}
	}

	return nil
}

// Convert the module into a node for the packaged template
func processModule(
	module *yaml.Node,
	outputNode *yaml.Node,
	t cft.Template,
	typeNode *yaml.Node,
	parent node.NodePair) (bool, error) {

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
	_, moduleResources, _ := s11n.GetMapValue(curNode, "Resources")

	if moduleResources == nil {
		return false, errors.New("expected the module to have a Resources section")
	}

	// Locate the Parameters: section in the module (might be nil)
	_, moduleParams, _ := s11n.GetMapValue(curNode, "Parameters")

	templateResource := parent.Value // The !!map node of the resource with Type !Rain::Module

	// Properties are the args that match module params
	_, templateProps, _ := s11n.GetMapValue(templateResource, "Properties")

	// Overrides have overridden values for module resources. Anything in a module can be overridden.
	_, overrides, _ := s11n.GetMapValue(templateResource, "Overrides")

	var fnForEachKey string
	var fnForEachName string
	var fnForEachItems *yaml.Node
	var fnForEach *yaml.Node
	var fnForEachSequence *yaml.Node
	var fnForEachLogicalId string

	// Fn::ForEach

	// Iterate through all resources and see if any of them start with Fn::ForEach
	for i := 0; i < len(moduleResources.Content); i += 2 {
		keyNode := moduleResources.Content[i]
		valueNode := moduleResources.Content[i+1]

		fnForEachKey = keyNode.Value

		if strings.HasPrefix(fnForEachKey, "Fn::ForEach") {
			//config.Debugf("Found a foreach: %v:\n%v", fnForEachKey, node.ToJson(valueNode))
			if valueNode.Kind != yaml.SequenceNode {
				return false, errors.New("expected Fn::ForEach to be a sequence")
			}
			if len(valueNode.Content) != 3 {
				return false, errors.New("expected Fn::ForEach to have 3 items")
			}
			// The Fn::ForEach intrinsic takes 3 array elements as input
			fnForEachName = valueNode.Content[0].Value
			fnForEachItems = node.Clone(valueNode.Content[1]) // TODO - Resolve refs
			feBody := valueNode.Content[2]

			// TODO: Items might be a Ref to a property set by the parent template
			// We need to try and resolve that Ref like any other

			if feBody.Kind != yaml.MappingNode {
				return false, errors.New("expected Fn::ForEach Body to be a mapping")
			}

			fnForEachLogicalId = feBody.Content[0].Value
			//feOutputMap := feBody.Content[1]
			//config.Debugf("LogicalId: %v\nOutputMap: %v", fnForEachLogicalId, feOutputMap)

			// Store this for later as we handle special cases for moduleExtension
			fnForEach = valueNode
			//config.Debugf("Fn::ForEach fnForEach: %v", node.ToJson(fnForEach))

			// Create node that looks like a regular ModuleExtenstion resource
			//moduleExtension := node.Clone(feOutputMap)
			// TODO: There is no ModuleExtension now

			//config.Debugf("Fn::ForEach moduleExtension: %v", node.ToJson(moduleExtension))

			// Make sure the parent template has Transform: AWS::LanguageExtensions
			docMap := t.Node.Content[0]
			_, transformNode, _ := s11n.GetMapValue(docMap, "Transform")
			if transformNode == nil {
				//config.Debugf("Adding Transform node")
				docMap.Content = append(docMap.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "Transform"})
				docMap.Content = append(docMap.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "AWS::LanguageExtensions"})
			}
		}
	}

	if fnForEach != nil {
		// Add the key for the Fn::ForEach from the module

		// We need to alter the name of the key to make sure it's unique, if the
		// module is used more than once in a template.
		// In the module if we have Fn::ForEach::MakeHandles:
		// and in the parent template the logical id is ForeachTest
		// then the key will be Fn::ForEach::ForeachTestMakeHandles:
		fixedKey := strings.Replace(fnForEachKey, "Fn::ForEach::", "Fn::ForEach::"+logicalId, 1)

		outputNode.Content = append(outputNode.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fixedKey,
		})

		//config.Debugf("foreach items: %v", node.ToJson(fnForEachItems))
		//config.Debugf("foreach name: %v", fnForEachName)

		fnForEachSequence = &yaml.Node{}
		fnForEachSequence.Kind = yaml.SequenceNode
		fnForEachSequence.Content = make([]*yaml.Node, 0)
		outputNode.Content = append(outputNode.Content, fnForEachSequence)

		fnForEachSequence.Content = append(fnForEachSequence.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: fnForEachName})

		// The second array element is the list of items to iterate over. It might be
		// a Ref to a parameter that supplies the values. Resolve the Ref.
		var resolvedItems *yaml.Node
		if fnForEachItems.Kind == yaml.SequenceNode {
			// TODO - Do we need to resolve individual items themselves? Will this be handled elsewhere?
			resolvedItems = fnForEachItems
		} else if fnForEachItems.Kind == yaml.MappingNode {
			if fnForEachItems.Content[0].Value == "Ref" {
				refName := fnForEachItems.Content[1].Value
				//config.Debugf("Fn::ForEach resolving items Ref %v", refName)
				_, p, _ := s11n.GetMapValue(moduleParams, refName)
				if p != nil {
					// Look up the value provided in the template props
					_, refval, _ := s11n.GetMapValue(templateProps, refName)
					if refval != nil {
						// This should be a comma separated value that we need to convert to a sequence
						resolvedItems = ConvertCsvToSequence(refval.Value)
					} else {
						// If it's not there, do we have a default in the module params?
						_, d, _ := s11n.GetMapValue(p, "Default")
						if d != nil {
							resolvedItems = ConvertCsvToSequence(d.Value)
						} else {
							// If not, leave it alone
							resolvedItems = fnForEachItems
						}
					}
				} else {
					// This is not a Ref to a module parameter.
					// TODO - Can this be a ref to something else in the module?
					// Leave it alone
					resolvedItems = fnForEachItems
				}

			} else {
				return false, errors.New("expected Fn::ForEach item map to be a Ref")
			}
		} else {
			return false, errors.New("expected Fn::ForEach items to be a sequence or a map")
		}
		//config.Debugf("resolvedItems: %v", node.ToJson(resolvedItems))

		fnForEachSequence.Content = append(fnForEachSequence.Content, resolvedItems)

		//config.Debugf("fnForEachSequence: %v", node.ToJson(fnForEachSequence))
	}

	// Get module resources and add them to the output
	for i, moduleResource := range moduleResources.Content {
		//config.Debugf("i = %v, resource = %v", i, moduleResource)
		if moduleResource.Kind != yaml.MappingNode {
			continue
		}
		name := moduleResources.Content[i-1].Value
		nameNode := node.Clone(moduleResources.Content[i-1])
		nameNode.Value = rename(logicalId, nameNode.Value)
		outputNode.Content = append(outputNode.Content, nameNode)
		clonedResource := node.Clone(moduleResource)

		// Get the overrides from the templates resource if there are any
		_, templateOverrides, _ := s11n.GetMapValue(overrides, name)

		// Clone attributes that are like Properties, and replace overridden values
		propLike := []string{"Properties", "CreationPolicy", "Metadata", "UpdatePolicy"}
		for _, pl := range propLike {
			_, plProps, _ := s11n.GetMapValue(moduleResource, pl)
			_, plTemplateProps, _ := s11n.GetMapValue(templateOverrides, pl)
			clonedProps := cloneAndReplaceProps(clonedResource, pl, plProps, plTemplateProps, moduleParams)
			if clonedProps == nil {
				// Was not present in the module or in the template, so skip it
				continue
			}
			if plProps != nil {
				// Get rid of what we cloned so we can replace it entirely
				node.RemoveFromMap(clonedResource, pl)
			}
			clonedResource.Content = append(clonedResource.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: pl})
			clonedResource.Content = append(clonedResource.Content, clonedProps)
		}

		// DeletionPolicy
		addPolicy(clonedResource, "DeletionPolicy", moduleResource, templateOverrides)

		// UpdateReplacePolicy
		addPolicy(clonedResource, "UpdateReplacePolicy", moduleResource, templateOverrides)

		// DependsOn is an array of scalars or a single scalar
		_, moduleDependsOn, _ := s11n.GetMapValue(moduleResource, "DependsOn")
		_, templateDependsOn, _ := s11n.GetMapValue(templateOverrides, "DependsOn")
		if moduleDependsOn != nil || templateDependsOn != nil {
			clonedResource.Content = append(clonedResource.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "DependsOn"})
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
			clonedResource.Content = append(clonedResource.Content, dependsOnValue)
		}

		// Resolve Refs in the module extension
		// Some refs are to other resources in the module
		// Other refs are to the module's parameters
		resolveRefs(clonedResource, moduleParams, moduleResources, logicalId, templateProps)

		// Add the Condition from the parent template
		_, parentCondition, _ := s11n.GetMapValue(templateOverrides, "Condition")
		if parentCondition != nil {
			clonedResource.Content = append(clonedResource.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "Condition"})
			clonedResource.Content = append(clonedResource.Content, node.Clone(parentCondition))
		}

		if fnForEachSequence != nil {
			// If the module has a ForEach extension, add it to the sequence instead
			//config.Debugf("Adding ext to the fnForEachSequence node")

			// The Fn::ForEach resource is a map, so we create that and append ext to it
			fnForEachMap := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
			// TODO
			newLogicalId := strings.Replace(fnForEachLogicalId, "ModuleExtension", logicalId, 1)
			fnForEachMap.Content = append(fnForEachMap.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: newLogicalId})
			fnForEachMap.Content = append(fnForEachMap.Content, clonedResource)

			// Add the map as the 3rd array element in the Fn::ForEach sequence
			fnForEachSequence.Content = append(fnForEachSequence.Content, fnForEachMap)

			//config.Debugf("outputNode after adding ext: %v", node.ToJson(outputNode))
		}

		// Resolve Conditions. Rain handles this differently, since a rain
		// module cannot have a Condition section. This value must be a module parameter
		// name, and the value must be set in the parent template as the name of
		// a Condition that is defined in the parent.
		_, condition, _ := s11n.GetMapValue(moduleResource, "Condition")
		if condition != nil {
			conditionErr := errors.New("a Condition in a rain module must be the name of " +
				"a Parameter that is set the name of a Condition in the parent template")
			// The value must be present in the module's parameters
			if condition.Kind != yaml.ScalarNode {
				return false, conditionErr
			}
			_, param, _ := s11n.GetMapValue(moduleParams, condition.Value)
			if param == nil {
				return false, conditionErr
			}
			_, conditionVal, _ := s11n.GetMapValue(templateProps, condition.Value)
			if conditionVal == nil {
				return false, conditionErr
			}
			if conditionVal.Kind != yaml.ScalarNode {
				return false, conditionErr
			}
			condition.Value = conditionVal.Value
		}

		/*
			// Modify referenced resource names
			_, dependsOn, _ := s11n.GetMapValue(clonedResource, "DependsOn")
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
		*/

		//resolveRefs(clonedResource, moduleParams, moduleResources, logicalId, templateProps)
		outputNode.Content = append(outputNode.Content, clonedResource)
	}

	return true, nil
}

// Type: !Rain::Module
func module(n *yaml.Node, root string, t cft.Template, parent node.NodePair) (bool, error) {

	if !Experimental {
		panic("You must add the --experimental arg to use the !Rain::Module directive")
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
	_, err = transform(&moduleNode, filepath.Dir(path), t, &parent)
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
	_, resourceNode, _ := s11n.GetMapValue(t.Node.Content[0], "Resources")
	if resourceNode == nil {
		return false, errors.New("expected template to have Resources")
	}

	// config.Debugf("resourceNode: %v", node.ToJson(resourceNode))

	// config.Debugf("outputNode: %v", node.ToJson(&outputNode))

	// Remove the original from the template
	err = node.RemoveFromMap(resourceNode, parent.Key.Value)
	if err != nil {
		return false, err
	}

	// Insert the transformed resource into the template
	resourceNode.Content = append(resourceNode.Content, outputNode.Content...)

	// config.Debugf("Returning from module")
	//config.Debugf("t.Node: %v", node.ToJson(t.Node))

	return true, nil

}
