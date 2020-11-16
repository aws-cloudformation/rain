// Package cft provides the Template type that models a CloudFormation template.
//
// The sub-packages of cft contain various tools for working with templates
package cft

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/aws-cloudformation/rain/internal/node"
)

// Template represents a CloudFormation template. The Template type
// is minimal for now but will likely grow new features as needed by rain.
type Template struct {
	yaml.Node
}

// Map returns the template as a map[string]interface{}
func (t Template) Map() map[string]interface{} {
	var out map[string]interface{}

	err := t.Decode(&out)
	if err != nil {
		panic(fmt.Errorf("Error converting template to map: %s", err))
	}

	return out
}

// Resolve returns a new template that has all values involving intrinsice functions resolved
// into concrete values. The parameters passed in will be used to populate parameters defined
// in the template or values of AWS pseudo parameters such as AWS::StackName
func (t Template) Resolve(params map[string]string) Template {
	n := node.Clone(&t.Node)

	resolve(n, params)

	return Template{*n}
}
