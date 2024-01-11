package ccdeploy

import (
	"encoding/json"
	"errors"
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
//	Fn::GetAtt
//
// Not Supported:
//
//	Fn::Base64
//	Fn::Cidr
//	Condition functions
//	Fn::FindInMap
//	Fn::ForEach
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

// TODO: What about intrinsics outside of Resources?

// resolveNode is a recursive function that resolves all
// intrinsics in the resource node and its children
func resolveNode(n *yaml.Node, resource *Resource) (*yaml.Node, error) {

	config.Debugf("resolveNode: %s", node.ToSJson(n))

	if n.Kind != yaml.MappingNode {
		return nil, errors.New("expected resource node to be a Mapping")
	}

	// We'll return a clone of the node, with intrinsics resolved
	retval := node.Clone(n)

	for i := 0; i < len(n.Content); i += 2 {
		mapkey := n.Content[i]
		mapval := n.Content[i+1]
		if mapkey.Kind == yaml.ScalarNode && mapkey.Value == "Ref" {
			config.Debugf("This is a Ref")
			refVal, err := resolveRef(mapval, resource)
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

		} else if mapkey.Kind == yaml.ScalarNode && mapkey.Value == "Fn::GetAtt" {

			config.Debugf("This is a GetAtt")
			getAttVal, err := resolveGetAtt(mapval, resource)
			if err != nil {
				return nil, err
			}

			// Same as with Ref above, we need to replace the node

			retval = &yaml.Node{Kind: yaml.ScalarNode, Value: getAttVal}

		} else if mapkey.Kind == yaml.ScalarNode && mapkey.Value == "Fn::Sub" {

			config.Debugf("This is a Sub")
			subVal, err := resolveSub(mapval, resource)
			if err != nil {
				return nil, err
			}

			// Same as with Ref above, we need to replace the node

			retval = &yaml.Node{Kind: yaml.ScalarNode, Value: subVal}

		} else if mapval.Kind == yaml.MappingNode {
			// Recurse on a child Mapping node
			config.Debugf("Recursing on child Mapping node")
			rn, err := resolveNode(mapval, resource)
			if err != nil {
				return nil, err
			}
			retval.Content[i+1] = rn
		}
	}

	return retval, nil
}

func resolveRef(refNode *yaml.Node, resource *Resource) (string, error) {
	if refNode.Kind != yaml.ScalarNode {
		return "", fmt.Errorf("ref Value is not a scalar for %v", resource.Name)
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
				return "", fmt.Errorf("resource %s does not have a Type?", refNode.Value)
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
			return "", fmt.Errorf("resource %s missing from global resource map", refNode.Value)
		}

		// Look at the resource model returned from when we deployed that resource
		config.Debugf("reffed id: %s,  model: %v", reffed.Identifier, reffed.Model)

		return reffed.Identifier, nil

	} else {
		config.Debugf("%v", err)
	}

	// Error if we can't find it anywhere
	return "", fmt.Errorf("cannot resolve %s", refNode.Value)
}

// resolveGetAtt resolves a node with Fn::GetAtt
func resolveGetAtt(n *yaml.Node, resource *Resource) (string, error) {
	if n.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("getAtt Value is not a Sequence for %v", resource.Name)
	}

	name := n.Content[0].Value
	attr := n.Content[1].Value

	config.Debugf("GetAtt %v.%v", name, attr)

	reffedResource, err := deployedTemplate.GetResource(name)
	if err != nil {
		return "", fmt.Errorf("can't find resource %s: %v", name, err)
	}
	config.Debugf("reffedResource: %v", reffedResource)

	// Get a reference to the Resource we deployed from the global map
	reffed, exists := resMap[name]
	if !exists {
		return "", fmt.Errorf("resource %s missing from global resource map", name)
	}

	// Look at the resource model returned from when we deployed that resource
	config.Debugf("reffed id: %s,  model: %v", reffed.Identifier, reffed.Model)

	// Parse the model to get the attribute
	var j map[string]any
	err = json.Unmarshal([]byte(reffed.Model), &j)
	if err != nil {
		return "", fmt.Errorf("unable to parse model: %v", err)
	}

	attrValue, exists := j[attr]
	if !exists {
		return "", fmt.Errorf("unable to find %s.%s in the deployed Model", name, attr)
	}

	return attrValue.(string), nil
}

// resolveSub resolves a node with Fn::Sub
func resolveSub(n *yaml.Node, resource *Resource) (string, error) {

	// A Sub will either have a Scalar string,
	// or a sequence of [String, Key:Val, Key:Val...]

	if n.Kind == yaml.ScalarNode {

		// ${X.Y} for GetAtt
		// AWS:: variables
		// Ref for single strings like ${MyParam} or ${MyBucket}
		// Map values
		// ${!Literal}
		return "", errors.New("not implemented")

	} else if n.Kind == yaml.SequenceNode {
		return "", errors.New("not implemented")
	} else {
		return "", errors.New("Expected a Scalar or a Sequence")
	}
}
