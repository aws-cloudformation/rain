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
	cloned := node.Clone(n)

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
					cloned.Content[i+1] = node.Clone(n.Content[i+1])
					cloned.Content[i+1].Value = refVal
				} else if n.Content[i+1].Kind == yaml.MappingNode {
					// Recurse on a child Mapping node
					return resolveNode(n.Content[i+1], resource)
				}
			}
		}

	case yaml.ScalarNode:
		config.Debugf("Scalar Node")

	case yaml.SequenceNode:
		config.Debugf("Sequence Node")

	default:
		config.Debugf("Unexpected Kind: %v", n.Kind)
		return cloned, nil
	}

	return cloned, nil
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

	// Check to see if it is a reference to another resource

	// Error if we can't find it anywhere
	return "", fmt.Errorf("Cannot resolve %s", refNode.Value)
}
