package ccdeploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/sts"
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
//	Fn::Sub
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
//	Fn::ToJsonString
//	Fn::Transform
func Resolve(resource *Resource) (*yaml.Node, error) {
	return resolveNode(resource.Node, resource)
}

const AWS_PREFIX = "AWS::"

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

	return resolveRefByName(refNode.Value, resource, make(map[string]string))
}

func resolvePseudoParam(name string) (string, error) {

	p := strings.Replace(name, AWS_PREFIX, "", 1)
	switch p {
	case "AccountId":
		return sts.GetAccountID()
	case "Region":
		return aws.Config().Region, nil
	case "NotificationARNs":
		// TODO: Can't return a string for this!
		return "", errors.New("unsupported: AWS::NotificationARNs")
	case "NoValue":
		// TODO: Needs special handling to remove nodes from the template
		return "", errors.New("unsupported: AWS::NoValue")
	case "Partition":
		region := aws.Config().Region
		if strings.HasPrefix(region, "us-gov") {
			return "aws-us-gov", nil
		}
		if strings.HasPrefix(region, "cn-") {
			return "aws-cn", nil
		}
		if strings.HasPrefix(region, "us-iso-") {
			return "aws-iso", nil
		}
		if strings.HasPrefix(region, "us-isob-") {
			return "aws-iso-b", nil
		}
		return "aws", nil
	case "StackId":
		return "", errors.New("unsupported: AWS::StackId")
	case "StackName":
		return "", errors.New("unsupported: AWS::StackName")
	case "URLSuffix":
		return "", errors.New("unsupported: AWS::URLSuffix")
	default:
		return "", fmt.Errorf("unexpected AWS::%s", p)
	}
}

func resolveRefByName(name string, resource *Resource, extra map[string]string) (string, error) {
	config.Debugf("resolveRefByName %v", name)

	// Handle pseudo-parameters
	if strings.HasPrefix(name, AWS_PREFIX) {
		return resolvePseudoParam(name)
	}

	// If extra has the name, use that. This comes from Fn::Sub
	if val, ok := extra[name]; ok {
		return val, nil
	}

	// Check to see if it is a Parameter
	p, _ := deployedTemplate.GetParameter(name)
	if p != nil {
		config.Debugf("Found parameter %v", node.ToSJson(p))

		// If the parameter value was supplied, use that value.
		// DeployConfig already takes default values into account.

		if pval, exists := templateConfig.GetParam(name); exists {
			config.Debugf("Supplied param value: %v", pval)
			return pval, nil
		}
	}

	config.Debugf("Did not find %s in Parameters, checking Resources", name)

	// Check to see if it is a reference to another resource
	reffedResource, err := deployedTemplate.GetResource(name)
	config.Debugf("reffedResource: %v", reffedResource)
	if err == nil {
		/*
			// Get the Type of the reffed resource
			_, t := s11n.GetMapValue(reffedResource, "Type")
			if t == nil {
				return "", fmt.Errorf("resource %s does not have a Type?", name)
			}
			reffedType := t.Value
		*/

		// Now we need to know, what does "Ref" mean for this resource type?
		// For now assume it's always primaryIdentifier

		// We don't need to query CCAPI to look at the schema for the type.
		// Because we already set resource.Identifier in deployment.go

		// Get a reference to the Resource we deployed from the global map
		reffed, exists := resMap[name]
		if !exists {
			return "", fmt.Errorf("resource %s missing from global resource map", name)
		}

		// Look at the resource model returned from when we deployed that resource
		config.Debugf("reffed id: %s,  model: %v", reffed.Identifier, reffed.Model)

		return reffed.Identifier, nil

	} else {
		config.Debugf("%v", err)
	}

	// Error if we can't find it anywhere
	return "", fmt.Errorf("cannot resolve %s", name)
}

// resolveGetAtt resolves a node with Fn::GetAtt
func resolveGetAtt(n *yaml.Node, resource *Resource) (string, error) {
	if n.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("getAtt Value is not a Sequence for %v", resource.Name)
	}

	name := n.Content[0].Value
	attr := n.Content[1].Value

	return resolveGetAttBy(name, attr, resource)
}

func resolveGetAttBy(name string, attr string, resource *Resource) (string, error) {

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

func resolveSubWords(words []word, resource *Resource, extra map[string]string) (string, error) {

	retval := ""
	prefix := ""

	for _, word := range words {
		switch word.t {
		case STR:
			retval += word.w
		case AWS:
			prefix = AWS_PREFIX
			fallthrough
		case REF:
			resolved, err := resolveRefByName(prefix+word.w, resource, extra)
			if err != nil {
				return "", err
			}
			retval += resolved
		case GETATT:
			left, right, found := strings.Cut(word.w, ".")
			if !found {
				return "", fmt.Errorf("unexpected GetAtt %s", word.w)
			}
			return resolveGetAttBy(left, right, resource)
		default:
			return "", fmt.Errorf("unexpected word type %v for %s", word.t, word.w)
		}
	}

	return retval, nil
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
		words, err := ParseSub(n.Value)
		if err != nil {
			return "", err
		}

		config.Debugf("Parsed \"%s\", got: %v", n.Value, words)

		return resolveSubWords(words, resource, make(map[string]string))
	} else if n.Kind == yaml.SequenceNode {
		sub := n.Content[0].Value
		if len(n.Content) != 2 {
			return "", fmt.Errorf("expected Sub %s sequence to have two elements, got %v", sub, len(n.Content))
		}
		if n.Content[1].Kind != yaml.MappingNode {
			return "", fmt.Errorf("expected Sub %s Content[1] to be a Mapping", sub)
		}
		words, err := ParseSub(sub)
		if err != nil {
			return "", err
		}

		config.Debugf("Parsed \"%s\", got: %v", sub, words)

		// Inline map values can be provided in array elements 1..n
		m := make(map[string]string)

		mapping := n.Content[1]
		for i := 0; i < len(mapping.Content); i += 2 {
			key := mapping.Content[i].Value
			subNode := mapping.Content[i+1]
			var subVal string
			if subNode.Kind == yaml.MappingNode {
				// Resolve this node like normal
				// Likely something like:
				// [ "${A}", "A", Map [ "Ref", "B" ] ]
				resolvedNode, err := resolveNode(subNode, resource)
				config.Debugf("Sub sequence mapping resolved: %v", node.ToSJson(resolvedNode))
				if resolvedNode.Kind != yaml.ScalarNode {
					return "", fmt.Errorf("expected resolved %s: %s to be a Scalar", sub, key)
				}
				subVal = resolvedNode.Value
				if err != nil {
					return "", err
				}
			} else if subNode.Kind == yaml.ScalarNode {
				subVal = subNode.Value
			} else {
				return "", fmt.Errorf("expected a Mapping or Scalar for Sub value %s: %s", sub, key)
			}
			m[key] = subVal
		}

		return resolveSubWords(words, resource, m)
	} else {
		return "", errors.New("expected a Scalar or a Sequence")
	}
}
