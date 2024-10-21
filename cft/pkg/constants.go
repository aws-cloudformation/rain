package pkg

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
)

// rainConstant parses a !Rain::Constant node
// Constants can be any type of YAML node, but if the constant is a string, it
// can be used in a Sub with ${Rain::ConstantName}. Otherwise use a directive.
// !Rain::Constant ConstantName. Constants are evaluated in order, so they can
// refer to other constants declared previously.
func rainConstant(ctx *directiveContext) (bool, error) {

	config.Debugf("Found a rain constant: %s", node.ToSJson(ctx.n))
	name := ctx.n.Content[1].Value
	val, ok := ctx.t.Constants[name]
	if !ok {
		return false, fmt.Errorf("rain constant %s not found", name)
	}

	*ctx.n = *val

	return true, nil
}
