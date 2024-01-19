// Package cft provides the Template type that models a CloudFormation template.
//
// The sub-packages of cft contain various tools for working with templates
package cft

import (
	"errors"
	"fmt"

	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
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

// AppendStateMap appends a "State" section to the template
func AppendStateMap(state Template) *yaml.Node {
	state.Node.Content[0].Content = append(state.Node.Content[0].Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "State"})
	stateMap := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
	state.Node.Content[0].Content = append(state.Node.Content[0].Content, stateMap)
	return stateMap
}

// Section represents a top level section of a template, like Resources
type Section string

const (
	AWSTemplateFormatVersion Section = "AWSTemplateFormatVersion"
	Resources                Section = "Resources"
	Description              Section = "Description"
	Metadata                 Section = "Metadata"
	Parameters               Section = "Parameters"
	Rules                    Section = "Rules"
	Mappings                 Section = "Mappings"
	Conditions               Section = "Conditions"
	Transform                Section = "Transform"
	Outputs                  Section = "Outputs"
)

// GetResource returns the yaml node for a resource by logical id
func (t Template) GetResource(name string) (*yaml.Node, error) {
	return t.GetNode(Resources, name)
}

// GetParameter returns the yaml node for a parameter by name
func (t Template) GetParameter(name string) (*yaml.Node, error) {
	return t.GetNode(Parameters, name)
}

// GetNode returns a yaml node by section and name
func (t Template) GetNode(section Section, name string) (*yaml.Node, error) {
	_, resMap := s11n.GetMapValue(t.Node.Content[0], string(section))
	if resMap == nil {
		return nil, fmt.Errorf("unable to locate the %s node", section)
	}
	// TODO: Some Sections are not Maps
	_, resource := s11n.GetMapValue(resMap, name)
	if resource == nil {
		return nil, fmt.Errorf("unable to locate %s %s", section, name)
	}
	return resource, nil
}

func (t Template) AddScalarSection(section Section, val string) error {
	if t.Node == nil {
		return errors.New("t.Node is nil")
	}
	if len(t.Node.Content) == 0 {
		return errors.New("missing Document Content")
	}
	m := t.Node.Content[0]
	node.Add(m, string(section), val)

	return nil
}
