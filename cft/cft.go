// Package cft provides the Template type that models a CloudFormation template.
//
// The sub-packages of cft contain various tools for working with templates
package cft

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Template represents a CloudFormation template. The Template type
// is minimal for now but will likely grow new features as needed by rain.
type Template struct {
	*yaml.Node
}

// TODO - We really need a convenient Template data structure
// that lets us easily access elements.
// t.Resources["MyResource"].Properties["MyProp"]
//
// Add a Model attribute to the struct and an Init function to populate it.
// t.Model.Resources

// Map returns the template as a map[string]interface{}
func (t Template) Map() map[string]interface{} {
	var out map[string]interface{}

	err := t.Decode(&out)
	if err != nil {
		panic(fmt.Errorf("error converting template to map: %s", err))
	}

	return out
}
