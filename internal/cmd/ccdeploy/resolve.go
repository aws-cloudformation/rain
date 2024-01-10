package ccdeploy

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"gopkg.in/yaml.v3"
)

// resolve resolves CloudFormation intrinsic functions
//
// It relies on dependent resources already having been deployed,
// so that we can query values with CCAPI.
//
// Supported:
//
//	Ref
//
// Not Supported:
//
//	Fn::Base64
//	Fn::Cidr
//	Condition functions
//	Fn::FindInMap
//	Fn::ForEach
//	Fn::GetAtt
//	Fn::GetAZs
//	Fn::ImportValue
//	Fn::Join
//	Fn::Length
//	Fn::Select
//	Fn::Split
//	Fn::Sub
//	Fn::ToJsonString
//	Fn::Transform
func Resolve(resource *Resource) (*yaml.Node, error) {
	return resolveNode(resource.Node, resource)
}

// resolveNode is a recursive function that resolves all
// intrinsics in the resource node and its children
func resolveNode(n *yaml.Node, resource *Resource) (*yaml.Node, error) {

	config.Debugf("resolveNode: %s", node.ToSJson(n))

	// We'll return a clone of the node, with intrinsics resolved
	retval := node.Clone(n)

	switch n.Kind {
	case yaml.MappingNode:
		config.Debugf("Mapping Node")
		for i, v := range n.Content {
			if i%2 == 0 {
				if v.Kind == yaml.ScalarNode && v.Value == "Ref" {
					config.Debugf("This is a Ref")
					refNode := n.Content[i+1]
					refVal, err := resolveRef(refNode, resource)
					if err != nil {
						return nil, err
					}

					/* We need to convert the entire Mapping node to a scalar

					n is:

					{
					  "Kind": "Mapping",
					  "Value": "",
					  "Content": [
						{
						  "Kind": "Scalar",
						  "Value": "Ref"
						},
						{
						  "Kind": "Scalar",
						  "Value": "aaa"
						}
					  ]
					}

					n needs to be:

					{
					  "Kind": "Scalar",
					  "Value": "aaa"
					}

					*/

					retval = &yaml.Node{Kind: yaml.ScalarNode, Value: refVal}

				} else if n.Content[i+1].Kind == yaml.MappingNode {
					// Recurse on a child Mapping node
					rn, err := resolveNode(n.Content[i+1], resource)
					if err != nil {
						return nil, err
					}
					retval.Content[i+1] = rn
				}
			}
		}

	case yaml.ScalarNode:
		config.Debugf("Scalar Node")

	case yaml.SequenceNode:
		config.Debugf("Sequence Node")

	default:
		config.Debugf("Unexpected Kind: %v", n.Kind)
	}

	return retval, nil
}

func resolveRef(refNode *yaml.Node, resource *Resource) (string, error) {
	if refNode.Kind != yaml.ScalarNode {
		return "", fmt.Errorf("Ref Value is not a scalar for %v", resource.Name)
	}

	// Now we have a name that we need to find.
	config.Debugf("refNode Value %v", refNode.Value)

	// Check to see if it is a Parameter
	p, _ := deployedTemplate.GetParameter(refNode.Value)
	if p != nil {
		config.Debugf("Found parameter %v", node.ToSJson(p))

		// If the parameter value was supplied, use that value.
		// DeployConfig already takes default values into account.

		if pval, exists := templateConfig.GetParam(refNode.Value); exists {
			config.Debugf("Supplied param value: %v", pval)
			return pval, nil
		}
	}

	config.Debugf("Did not find %s in Parameters, checking Resources", refNode.Value)

	// Check to see if it is a reference to another resource
	reffedResource, err := deployedTemplate.GetResource(refNode.Value)
	config.Debugf("reffedResource: %v", reffedResource)
	if err == nil {
		/*
			// Get the Type of the reffed resource
			_, t := s11n.GetMapValue(reffedResource, "Type")
			if t == nil {
				return "", fmt.Errorf("Resource %s does not have a Type?", refNode.Value)
			}
			reffedType := t.Value
		*/

		// Now we need to know, what does "Ref" mean for this resource type?
		// For now assume it's always primaryIdentifier

		// We don't need to query CCAPI to look at the schema for the type.
		// Because we already set resource.Identifier in deployment.go

		// Get a reference to the Resource we deployed from the global map
		reffed, exists := resMap[refNode.Value]
		if !exists {
			return "", fmt.Errorf("Resource %s missing from global resource map", refNode.Value)
		}

		// Look at the resource model returned from when we deployed that resource
		config.Debugf("reffed id: %s,  model: %v", reffed.Identifier, reffed.Model)

		return reffed.Identifier, nil

	} else {
		config.Debugf("%v", err)
	}

	// Error if we can't find it anywhere
	return "", fmt.Errorf("Cannot resolve %s", refNode.Value)
}
