package pkg

import (
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"gopkg.in/yaml.v3"
)

// rainConstant parses a !Rain::Constant node
// Constants can be any type of YAML node, but if the constant is a string, it
// can be used in a Sub with ${Rain::ConstantName}. Otherwise use a directive.
// !Rain::Constant ConstantName. Constants are evaluated in order, so they can
// refer to other constants declared previously.
func rainConstant(ctx *directiveContext) (bool, error) {

	config.Debugf("Found a rain constant: %s", node.ToSJson(ctx.n))

	// TODO
	*ctx.n = yaml.Node{Kind: yaml.ScalarNode, Value: "TODO"}

	return true, nil
}
