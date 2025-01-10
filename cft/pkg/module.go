// This file implements !Rain::Module
package pkg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

var HasModules bool

const (
	Rain                = "Rain"
	Metadata            = "Metadata"
	IfParam             = "IfParam"
	IfNotParam          = "IfNotParam"
	Overrides           = "Overrides"
	DependsOn           = "DependsOn"
	Properties          = "Properties"
	CreationPolicy      = "CreationPolicy"
	UpdatePolicy        = "UpdatePolicy"
	DeletionPolicy      = "DeletionPolicy"
	UpdateReplacePolicy = "UpdateReplacePolicy"
	Condition           = "Condition"
	Default             = "Default"
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

			// Overwrite anything hard coded into the module that is
			// present in the parent template
			for j, mprop := range props.Content {
				if tprop.Value == mprop.Value && i%2 == 0 && j%2 == 0 {
					clonedNode := node.Clone(templateProps.Content[i+1])
					// config.Debugf("original: %s", node.ToSJson(templateProps.Content[i+1]))
					// config.Debugf("clonedNode: %s", node.ToSJson(clonedNode))
					merged := node.MergeNodes(props.Content[j+1], clonedNode)
					// config.Debugf("merged: %s", node.ToSJson(merged))
					props.Content[j+1] = merged

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

// Add DeletionPolicy, UpdateReplacePolicy, and Condition
func addScalarAttribute(out *yaml.Node, name string, moduleResource *yaml.Node, templateOverrides *yaml.Node) {
	_, templatePolicy, _ := s11n.GetMapValue(templateOverrides, name)
	_, modulePolicy, _ := s11n.GetMapValue(moduleResource, name)
	if modulePolicy != nil {
		node.RemoveFromMap(out, name)
	}
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

// Common context needed to resolve Refs in the module.
// This is all the common stuff that is the same for this module.
type refctx struct {
	// The module's Parameters
	moduleParams *yaml.Node

	// The parent template's Properties
	templateProps *yaml.Node

	// The node we're writing to for output to the resulting template
	outNode *yaml.Node

	// The logical id of the resource in the parent template
	logicalId string

	// The module's Resources map
	moduleResources *yaml.Node

	// Template property overrides map for the resource
	// TODO: Not necessary? We don't look anything up here...
	overrides *yaml.Node

	// The module's Constants from the Rain section
	constants map[string]*yaml.Node
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

// Resolve a Ref.
// parentName is the name of the Property with the Ref in it.
// prop is the Scalar node with the value for the Ref.
// The output node is modified by this function (or the prop, which is part of the output)
func resolveModuleRef(parentName string, prop *yaml.Node, sidx int, ctx *refctx) error {

	// MyProperty: !Ref NameOfParam
	//
	// MyProperty is the parentName
	// NameOfParam is prop.Value

	moduleParams := ctx.moduleParams
	templateProps := ctx.templateProps
	outNode := ctx.outNode
	logicalId := ctx.logicalId
	moduleResources := ctx.moduleResources

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
func resolveModuleSub(parentName string, prop *yaml.Node, sidx int, ctx *refctx) error {

	moduleParams := ctx.moduleParams
	templateProps := ctx.templateProps
	logicalId := ctx.logicalId
	moduleResources := ctx.moduleResources

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
		// This should not be necessary since we process Rain constants earlier
		//case parse.RAIN:
		//	// Replace ${Rain::ConstantName} with template constant value
		//	if ctx.constants == nil {
		//		return fmt.Errorf("no Rain Constants section, looking for %s", word.W)
		//	}
		//	if c, ok := ctx.constants[word.W]; ok {
		//		sub += c.Value
		//	} else {
		//		if len(ctx.constants) == 0 {
		//			config.Debugf("Constants are empty")
		//		}
		//		for k, v := range ctx.constants {
		//			config.Debugf("Constant %s: %s", k, v.Value)
		//		}
		//		return fmt.Errorf("unable to find Rain constant %s", word.W)
		//	}
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
	replaceProp(prop, parentName, newProp, ctx.outNode, sidx)

	return nil
}

// Recursive function to find all refs in properties
// Also handles DeletionPolicy, UpdateRetainPolicy
// If sidx is > -1, this prop is in a sequence
func renamePropRefs(parentName string, propName string, prop *yaml.Node, sidx int, ctx *refctx) error {

	logicalId := ctx.logicalId

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
			if err := resolveModuleRef(parentName, prop, sidx, ctx); err != nil {
				return fmt.Errorf("resolving module ref %s: %v", parentName, err)
			}
		} else if propName == "Fn::Sub" {

			if err := resolveModuleSub(parentName, prop, sidx, ctx); err != nil {
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
				result := renamePropRefs(propName, p.Value, prop.Content[i], i, ctx)
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
				result := renamePropRefs(propName, p.Value, prop.Content[i+1], passSidx, ctx)
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

// Convert !Ref values
func resolveRefs(ctx *refctx) error {

	outNode := ctx.outNode

	// Replace references to the module's parameters with the value supplied
	// by the parent template. Rename refs to other resources in the module.
	propLikes := []string{Properties, Metadata}
	for _, propLike := range propLikes {
		_, outNodeProps, _ := s11n.GetMapValue(outNode, propLike)
		if outNodeProps != nil {
			for i, prop := range outNodeProps.Content {
				if i%2 == 0 {
					propName := prop.Value
					err := renamePropRefs(propName, propName, outNodeProps.Content[i+1], -1, ctx)
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
		_, policyNode, _ := s11n.GetMapValue(outNode, policy)
		if policyNode != nil {
			err := renamePropRefs(policy, policy, policyNode, -1, ctx)
			if err != nil {
				return fmt.Errorf("unable to resolve refs for %v, %v", policy, err)
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
	parent node.NodePair,
	moduleConstants map[string]*yaml.Node) (bool, error) {

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
	_, templateProps, _ := s11n.GetMapValue(templateResource, Properties)

	// Overrides have overridden values for module resources. Anything in a module can be overridden.
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
				return false, fmt.Errorf("override not found: %s", override.Value)
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
							return false,
								fmt.Errorf("cannot override module parameter %s",
									overrideProp.Value)
						}
					}
				}
			}
		}
	}

	fe, err := handleForEach(moduleResources, t, logicalId, outputNode,
		moduleParams, templateProps)
	if err != nil {
		return false, err
	}

	// Get module resources and add them to the output
	for i, moduleResource := range moduleResources.Content {
		if moduleResource.Kind != yaml.MappingNode {
			continue
		}
		name := moduleResources.Content[i-1].Value

		// Check to see if there is a Rain attribute in the Metadata.
		// If so, check conditionals like IfParam
		metadata := s11n.GetMap(moduleResource, Metadata)
		if metadata != nil {
			if rainMetadata, ok := metadata[Rain]; ok {
				if omitIfs(rainMetadata, moduleParams, templateProps, moduleResource) {
					continue
				}
			}
		}

		nameNode := node.Clone(moduleResources.Content[i-1])
		nameNode.Value = rename(logicalId, nameNode.Value)
		outputNode.Content = append(outputNode.Content, nameNode)
		clonedResource := node.Clone(moduleResource)

		// Get the overrides from the templates resource if there are any
		var templateOverrides *yaml.Node
		if overrides != nil {
			_, templateOverrides, _ = s11n.GetMapValue(overrides, name)
		}

		// Clone attributes that are like Properties, and replace overridden values
		propLike := []string{Properties, CreationPolicy, Metadata, UpdatePolicy}
		for _, pl := range propLike {
			_, plProps, _ := s11n.GetMapValue(moduleResource, pl)
			_, plTemplateProps, _ := s11n.GetMapValue(templateOverrides, pl)
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
		addScalarAttribute(clonedResource, DeletionPolicy, moduleResource, templateOverrides)

		// UpdateReplacePolicy
		addScalarAttribute(clonedResource, UpdateReplacePolicy, moduleResource, templateOverrides)

		// Condition
		addScalarAttribute(clonedResource, Condition, moduleResource, templateOverrides)

		// DependsOn is an array of scalars or a single scalar
		_, moduleDependsOn, _ := s11n.GetMapValue(moduleResource, DependsOn)
		_, templateDependsOn, _ := s11n.GetMapValue(templateOverrides, DependsOn)
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

		/*
			// Add the Condition from the parent template
			_, parentCondition, _ := s11n.GetMapValue(templateOverrides, Condition)
			if parentCondition != nil {
				clonedResource.Content = append(clonedResource.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: Condition})
				clonedResource.Content = append(clonedResource.Content, node.Clone(parentCondition))
			}
		*/

		// Resolve Refs in the module
		// Some refs are to other resources in the module
		// Other refs are to the module's parameters
		ctx := &refctx{
			moduleParams:    moduleParams,
			templateProps:   templateProps,
			outNode:         clonedResource,
			logicalId:       logicalId,
			moduleResources: moduleResources,
			overrides:       templateOverrides,
			constants:       moduleConstants,
		}
		err := resolveRefs(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to resolve refs: %v", err)
		}

		if fe != nil && fe.fnForEachSequence != nil {
			// If the module has a ForEach extension, add it to the sequence instead

			// The Fn::ForEach resource is a map, so we create that and append outNode to it
			fnForEachMap := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
			// TODO
			newLogicalId := strings.Replace(fe.fnForEachLogicalId, "ModuleExtension", logicalId, 1)
			fnForEachMap.Content = append(fnForEachMap.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: newLogicalId})
			fnForEachMap.Content = append(fnForEachMap.Content, clonedResource)

			// Add the map as the 3rd array element in the Fn::ForEach sequence
			fe.fnForEachSequence.Content = append(fe.fnForEachSequence.Content, fnForEachMap)

		}

		outputNode.Content = append(outputNode.Content, clonedResource)
	}

	return true, nil
}

func checkPackageAlias(t cft.Template, uri string) *cft.PackageAlias {
	tokens := strings.Split(uri, "/")
	if len(tokens) > 1 {
		// See if this is one of the template package aliases
		if t.Packages != nil {
			if p, ok := t.Packages[tokens[0]]; ok {
				return p
			}
		}
	}
	return nil
}

// Type: !Rain::Module
func module(ctx *directiveContext) (bool, error) {

	n := ctx.n
	root := ctx.rootDir
	t := ctx.t
	parent := ctx.parent
	templateFiles := ctx.fs

	if !Experimental {
		panic("You must add the --experimental arg to use the !Rain::Module directive")
	}

	if len(n.Content) != 2 {
		return false, errors.New("expected !Rain::Module <URI>")
	}

	HasModules = true

	uri := n.Content[1].Value
	var content []byte
	var err error
	var path string
	var newRootDir string

	baseUri := ctx.baseUri

	// Check to see if this is an alias like "alias/foo.yaml"
	packageAlias := checkPackageAlias(t, uri)
	isZip := false
	if packageAlias != nil {
		config.Debugf("Found package alias: %+v", packageAlias)
		path := strings.Replace(uri, packageAlias.Alias+"/", "", 1)
		config.Debugf("path is %s", path)
		if strings.HasSuffix(packageAlias.Location, ".zip") {
			// Unzip, verify hash if there is one, and put the files in memory
			isZip = true
			content, err = DownloadFromZip(packageAlias.Location, packageAlias.Hash, path)
			if err != nil {
				return false, err
			}
		} else {
			uri = strings.Replace(uri, packageAlias.Alias, packageAlias.Location, 1)
			config.Debugf("uri is now %s", uri)
			config.Debugf("baseUri is %s", baseUri)
		}
	}

	// Is this a local file or a URL or did we already unzip a package?
	if isZip {
		config.Debugf("Got content from a zipped module package: %s", string(content))
	} else if strings.HasPrefix(uri, "https://") {

		content, err = downloadModule(uri)
		if err != nil {
			return false, err
		}

		// Once we see a URL instead of a relative local path,
		// we need to remember the base URL so that we can
		// fix relative paths in any referenced modules.

		// Strip the file name from the uri
		urlParts := strings.Split(uri, "/")
		baseUri = strings.Join(urlParts[:len(urlParts)-1], "/")

	} else {
		if baseUri != "" {
			// If we have a base URL, prepend it to the relative path
			uri = baseUri + "/" + uri
			content, err = downloadModule(uri)
			if err != nil {
				return false, err
			}
		} else if templateFiles != nil {
			// Read from the embedded file system (for the build -r command)
			path, err = expectString(n)
			if err != nil {
				return false, err
			}
			// We have to hack this since embed doesn't understand "path/../"
			embeddedPath := strings.Replace(root, "../", "", 1) +
				"/" + strings.Replace(path, "../", "", 1)

			content, err = templateFiles.ReadFile(embeddedPath)
			if err != nil {
				return false, err
			}
			newRootDir = filepath.Dir(embeddedPath)
		} else {
			// Read the local file
			path := uri
			if !filepath.IsAbs(path) {
				path = filepath.Join(root, path)
			}

			info, err := os.Stat(path)
			if err != nil {
				return false, err
			}

			if info.IsDir() {
				return false, fmt.Errorf("'%s' is a directory", path)
			}

			content, err = os.ReadFile(path)
			if err != nil {
				return false, err
			}
			newRootDir = filepath.Dir(path)
		}
	}

	// Parse the file
	var moduleNode yaml.Node
	err = yaml.Unmarshal(content, &moduleNode)
	if err != nil {
		return false, err
	}

	err = parse.NormalizeNode(&moduleNode)
	if err != nil {
		return false, err
	}

	// Figure out parent nodes to handle nested modules
	var newParent node.NodePair
	if parent.Parent != nil && parent.Parent.Value != nil {
		newParent = node.GetParent(n, parent.Parent.Value, nil)
		newParent.Parent = &parent
	}

	// Treat the module as a template
	moduleAsTemplate := cft.Template{Node: &moduleNode}

	// Read things like Constants
	processRainSection(&moduleAsTemplate)

	if moduleAsTemplate.Constants != nil {
		replaceTemplateConstants(moduleAsTemplate.Node, moduleAsTemplate.Constants)
	}

	_, err = transform(&transformContext{
		nodeToTransform: &moduleNode,
		rootDir:         newRootDir,
		t:               moduleAsTemplate,
		parent:          &newParent,
		fs:              ctx.fs,
		baseUri:         baseUri,
	})
	if err != nil {
		return false, err
	}

	// Create a new node to represent the processed module
	var outputNode yaml.Node
	_, err = processModule(&moduleNode, &outputNode, t, n, parent, moduleAsTemplate.Constants)
	if err != nil {
		config.Debugf("processModule error: %v, moduleNode: %s", err, node.ToSJson(&moduleNode))
		return false, fmt.Errorf("failed to process module %s: %v", uri, err)
	}

	// Find the resource node in the template
	_, resourceNode, _ := s11n.GetMapValue(t.Node.Content[0], "Resources")
	if resourceNode == nil {
		return false, errors.New("expected template to have Resources")
	}

	// Remove the original from the template
	err = node.RemoveFromMap(resourceNode, parent.Key.Value)
	if err != nil {
		config.Debugf("err removing original: %s\n%v",
			parent.Key.Value, node.ToSJson(resourceNode))
		return false, fmt.Errorf("can't remove original from template: %v", err)
	}

	// Insert the transformed resource into the template
	resourceNode.Content = append(resourceNode.Content, outputNode.Content...)

	return true, nil

}
