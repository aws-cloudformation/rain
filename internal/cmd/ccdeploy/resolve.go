package ccdeploy

import (
	"github.com/aws-cloudformation/rain/internal/node"
	"gopkg.in/yaml.v3"
)

// resolve resolves CloudFormation intrinsic functions
//
// Supported:
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
//	Ref
func resolve(resource *Resource) (*yaml.Node, error) {
	return node.Clone(resource.Node), nil

	// TODO
}
