// The file has functions related to resolving refs in modules
package pkg

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// Resolve a Ref.  parentName is the name of the Property with the Ref in it.
// prop is the Scalar node with the value for the Ref.  The output node is
// modified by this function (or the prop, which is part of the output)
func (module *Module) ResolveRef(parentName string, prop *yaml.Node, sidx int, outNode *yaml.Node) error {

	// MyProperty: !Ref NameOfParam
	//
	// MyProperty is the parentName
	// NameOfParam is prop.Value

	moduleParams := module.ParametersNode
	templateProps := module.Config.PropertiesNode
	logicalId := module.Config.Name
	moduleResources := module.ResourcesNode

	refFoundInParams := false

	if moduleParams != nil {
		// Find the module parameter that matches the !Ref
		_, param, _ := s11n.GetMapValue(moduleParams, prop.Value)
		if param != nil {
			// We need to get the parameter value from the parent template.
			// Module params are set by the parent template resource properties.
			//
			// For example:
			//
			// The module has this section:
			//
			// Parameters:
			//   Foo:
			//     Type: String
			//
			// And the parent template has this:
			//
			// MyResource:
			//   Type: !Rain::Module "this-module.yaml"
			//   Properties:
			//     Foo: bar
			//
			// Inside the module, we replace !Ref Foo with bar

			// Look for this property name in the parent template
			_, parentVal, _ := s11n.GetMapValue(templateProps, prop.Value)
			if parentVal == nil {
				// Check to see if there is a Default
				_, mParam, _ := s11n.GetMapValue(moduleParams, prop.Value)
				if mParam != nil {
					_, defaultNode, _ := s11n.GetMapValue(mParam, Default)
					if defaultNode != nil {
						parentVal = defaultNode
					}
				}

				// If we didn't find a parent template prop or a default, fail
				if parentVal == nil {
					return fmt.Errorf("did not find %v in parent template Properties",
						prop.Value)
				}
			}

			replaceProp(prop, parentName, parentVal, outNode, sidx)

			refFoundInParams = true
		}
	}
	if !refFoundInParams {
		// Look for a resource in the module
		_, resource, _ := s11n.GetMapValue(moduleResources, prop.Value)
		if resource == nil {
			// If we can't find the Ref, leave it alone and assume it's
			// expected to be in the parent template to be resolved at deploy
			// time. This is sort of cheating. It means you can write a module
			// that has to know about its parent. For example, if you put !Ref
			// Foo in the module, and Foo appears nowhere in the module, we
			// assume it will show up in the parent template. For some use
			// cases, it makes sense to allow this and not consider it an error.
			return nil
		}
		fixedName := rename(logicalId, prop.Value)
		prop.Value = fixedName
	}
	return nil
}

// Resolve a Sub string in a module.
//
// Sub strings can contain several types of variables.
// We leave intrinsics like ${AWS::Region} alone.
// ${Foo} is treated like a Ref to Foo
// ${Foo.Bar} is treated like a GetAtt.
//
// Shares logic with resolveModuleRef, but operates on substrings,
// which must resolve to strings and not objects.
//
// prop.Value is the Sub string
// sidx is the sequence index if it's > -1
// ctx.outNode will be modified to replace prop.Value with the references
func (module *Module) ResolveSub(parentName string, prop *yaml.Node, sidx int, outNode *yaml.Node) error {

	moduleParams := module.ParametersNode
	templateProps := module.Config.PropertiesNode
	logicalId := module.Config.Name
	moduleResources := module.ResourcesNode

	refFoundInParams := false

	words, err := parse.ParseSub(prop.Value, true)
	if err != nil {
		return err
	}

	sub := ""
	needSub := false // If we can fully resolve everything, we can remove the !Sub
	for _, word := range words {
		switch word.T {
		case parse.STR:
			sub += word.W
		case parse.AWS:
			sub += "${AWS::" + word.W + "}"
			needSub = true
		case parse.REF:
			resolved := fmt.Sprintf("${%s}", word.W)

			// Look for the name in module params
			if moduleParams != nil {
				// Find the module parameter that matches the !Ref
				_, param, _ := s11n.GetMapValue(moduleParams, word.W)
				if param != nil {
					_, parentVal, _ := s11n.GetMapValue(templateProps, word.W)
					if parentVal == nil {
						return fmt.Errorf("did not find %v in parent template Properties", prop.Value)
					}
					if parentVal.Kind == yaml.MappingNode {
						// In the parent template, the property is a Sub
						// This would need to resolve to a string so assume a len of 2
						if len(parentVal.Content) == 2 {
							needSub = true
							if parentVal.Content[0].Value == "Ref" {
								resolved = fmt.Sprintf("${%s}", parentVal.Content[1].Value)
							} else {
								resolved = parentVal.Content[1].Value
							}
						}
					} else {
						// It's a string
						resolved = parentVal.Value
					}
					refFoundInParams = true
				} else {
					needSub = true
				}
			} else {
				needSub = true
			}
			if !refFoundInParams {
				// Look for a resource in the module
				_, resource, _ := s11n.GetMapValue(moduleResources, word.W)
				if resource != nil {
					resolved = rename(logicalId, word.W)
				} else {
					needSub = true
				}
			}

			// If we didn't change the word, it is either an intrinsic like AWS::Region or
			// a value that is expected to be in the parent template, which is up to the user

			sub += resolved
		case parse.GETATT:
			// All we do here is fix the left part of the GetAtt
			// ${Foo.Bar} becomes ${NameFoo.Bar} where Name is the logicalId
			needSub = true
			left, right, found := strings.Cut(word.W, ".")
			if !found {
				return fmt.Errorf("unexpected GetAtt %s", word.W)
			}
			_, resource, _ := s11n.GetMapValue(moduleResources, left)
			if resource != nil {
				left = rename(logicalId, left)
			}
			sub += fmt.Sprintf("${%s.%s}", left, right)
			needSub = true
		default:
			return fmt.Errorf("unexpected word type %v for %s", word.T, word.W)
		}
	}

	// Put the sub back if there were any unresolved variables
	var newProp *yaml.Node
	if needSub && sidx < 0 {
		newProp = &yaml.Node{Kind: yaml.MappingNode, Value: parentName}
		newProp.Content = make([]*yaml.Node, 0)
		newProp.Content = append(newProp.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "Fn::Sub"})
		newProp.Content = append(newProp.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: sub})
	} else {
		newProp = &yaml.Node{Kind: yaml.ScalarNode, Value: sub}
	}

	// Replace the prop in the output node
	replaceProp(prop, parentName, newProp, outNode, sidx)

	return nil
}

// Recursive function to find all refs in properties
// Also handles DeletionPolicy, UpdateRetainPolicy
// If sidx is > -1, this prop is in a sequence
func (module *Module) RenamePropRefs(parentName string, propName string, prop *yaml.Node, sidx int, outNode *yaml.Node) error {

	logicalId := module.Config.Name

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

	if prop.Kind == yaml.ScalarNode {
		if propName == "Ref" {
			if err := module.ResolveRef(parentName, prop, sidx, outNode); err != nil {
				return fmt.Errorf("resolving module ref %s: %v", parentName, err)
			}
		} else if propName == "Fn::Sub" {
			if err := module.ResolveSub(parentName, prop, sidx, outNode); err != nil {
				return fmt.Errorf("resolving module sub %s: %v", parentName, err)
			}

			// TODO: Also handle Seq Subs
		}
	} else if prop.Kind == yaml.SequenceNode {
		if propName == "Fn::GetAtt" {
			// Convert !GetAtt Name.Property to !GetAtt LogicalId.Property
			fixedName := rename(logicalId, prop.Content[0].Value)
			prop.Content[0].Value = fixedName
		} else {
			// Recurse over array elements
			for i, p := range prop.Content {
				// propName is blank so the next parentName is blank
				result := module.RenamePropRefs(propName, p.Value, prop.Content[i], i, outNode)
				if result != nil {
					return fmt.Errorf("recursing over array %s: %v", parentName, result)
				}
			}
		}
	} else if prop.Kind == yaml.MappingNode {
		// Iterate over all map elements and recurse on the contents
		for i, p := range prop.Content {
			if i%2 == 0 {

				// Don't pass sidx through if we're in a child node of the sequence
				passSidx := sidx
				if propName != "" {
					passSidx = -1
				}
				result := module.RenamePropRefs(propName, p.Value, prop.Content[i+1], passSidx, outNode)
				if result != nil {
					return fmt.Errorf("recursing over mapping node %s: %v", propName, result)
				}
			}
		}
	} else {
		return fmt.Errorf("unexpected prop Kind: %v", prop.Kind)
	}

	return nil
}

// Resolve resolves Ref, Sub, and GetAtt in the node,
// using module config Properties, Parameter defaults, and
// other Resource names within the module.
func (module *Module) Resolve(n *yaml.Node) error {

	// Replace references to the module's parameters with the value supplied
	// by the parent template. Rename refs to other resources in the module.
	propLikes := []string{Properties, Metadata}
	for _, propLike := range propLikes {
		_, outNodeProps, _ := s11n.GetMapValue(n, propLike)
		if outNodeProps != nil {
			for i, prop := range outNodeProps.Content {
				if i%2 == 0 {
					propName := prop.Value
					err := module.RenamePropRefs(propName, propName, outNodeProps.Content[i+1], -1, n)
					if err != nil {
						return fmt.Errorf("unable to resolve refs for %s %v: %v",
							propLike, propName, err)
					}
				}
			}
		}
	}

	// DeletionPolicy, UpdateReplacePolicy, Condition
	policies := []string{DeletionPolicy, UpdateReplacePolicy, Condition}
	for _, policy := range policies {
		_, policyNode, _ := s11n.GetMapValue(n, policy)
		if policyNode != nil {
			err := module.RenamePropRefs(policy, policy, policyNode, -1, n)
			if err != nil {
				return fmt.Errorf("unable to resolve refs for %v, %v", policy, err)
			}
		}
	}

	return nil
}

// Rename a resource defined in the module to add the template resource name
func rename(logicalId string, resourceName string) string {
	return logicalId + resourceName
}

func replaceProp(prop *yaml.Node, parentName string, v *yaml.Node, outNode *yaml.Node, sidx int) error {

	if sidx > -1 {
		// The node is a sequence element

		newVal := node.Clone(v)

		if v.Kind == yaml.MappingNode {
			parentNode := node.GetParent(prop, outNode, nil)
			*parentNode.Value = *newVal
		} else {
			parentNode := node.GetParent(prop, outNode, nil)
			if parentNode.Key != nil {
				*parentNode.Value = *newVal
			} else {
				*prop = *newVal
			}
		}
		return nil
	}

	// We can't just set prop.Value, since we would end up with
	// Prop: !Ref Value instead of just Prop: Value. Get the
	// property's parent and set the entire map value for the
	// property

	// Get the map parent within the output node we created
	refMap := node.GetParent(prop, outNode, nil)
	if refMap.Value == nil {
		return fmt.Errorf("could not find parent for %v", prop)
	}
	propParentPair := node.GetParent(refMap.Value, outNode, nil)

	// Create a new node to replace what's defined in the module
	newValue := node.Clone(v)

	node.SetMapValue(propParentPair.Value, parentName, newValue)

	return nil
}
