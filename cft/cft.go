// Package cft provides the Template type that models a CloudFormation template.
//
// The sub-packages of cft contain various tools for working with templates
package cft

import (
	"errors"
	"fmt"
	"slices"

	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// PackageAlias is an alias to a module package location
// A Rain package is a directory of modules, which are single yaml files.
// See the main README for more
type PackageAlias struct {
	// Alias is a simple string like "aws"
	Alias string

	// Location is the URI where the package is stored
	Location string

	// Hash is an optional hash for zipped packages hosted on a URL
	Hash string
}

// Template represents a CloudFormation template. The Template type
// is minimal for now but will likely grow new features as needed by rain.
type Template struct {
	Node *yaml.Node

	Constants map[string]*yaml.Node
	Packages  map[string]*PackageAlias
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

	err := t.Node.Decode(&out)
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
	State                    Section = "State"
	Rain                     Section = "Rain"
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
	_, resMap, _ := s11n.GetMapValue(t.Node.Content[0], string(section))
	if resMap == nil {
		config.Debugf("GetNode t.Node: %s", node.ToSJson(t.Node))
		return nil, fmt.Errorf("unable to locate the %s node", section)
	}
	// TODO: Some Sections are not Maps
	_, resource, _ := s11n.GetMapValue(resMap, name)
	if resource == nil {
		return nil, fmt.Errorf("unable to locate %s %s", section, name)
	}
	return resource, nil
}

// AddScalarSection adds a section like Description to the template
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

// AddMapSection adds a section like Resources to the template
func (t Template) AddMapSection(section Section) (*yaml.Node, error) {

	if t.Node == nil {
		return nil, errors.New("t.Node is nil")
	}
	if len(t.Node.Content) == 0 {
		return nil, errors.New("missing Document Content")
	}

	m := t.Node.Content[0]
	return node.AddMap(m, string(section)), nil
}

// GetSection returns the yaml node for the section
func (t Template) GetSection(section Section) (*yaml.Node, error) {
	if t.Node == nil {
		return nil, fmt.Errorf("unable to get section because t.Node is nil")
	}
	_, s, _ := s11n.GetMapValue(t.Node.Content[0], string(section))
	if s == nil {
		config.Debugf("GetSection t.Node: %s", node.ToSJson(t.Node))
		return nil, fmt.Errorf("unable to locate the %s node", section)
	}
	return s, nil
}

// RemoveSection removes a section node from the template
func (t Template) RemoveSection(section Section) error {
	return node.RemoveFromMap(t.Node.Content[0], string(Rain))
}

// GetTypes returns all unique type names for resources in the template
func (t Template) GetTypes() ([]string, error) {
	resources, err := t.GetSection(Resources)
	if err != nil {
		return nil, err
	}
	retval := make([]string, 0)

	for i := 0; i < len(resources.Content); i += 2 {
		logicalId := resources.Content[i].Value
		resource := resources.Content[i+1]
		_, typ, _ := s11n.GetMapValue(resource, "Type")
		if typ == nil {
			return nil, fmt.Errorf("expected %s to have Type", logicalId)
		}
		if !slices.Contains(retval, typ.Value) {
			retval = append(retval, typ.Value)
		}
	}

	return retval, nil
}

type Resource struct {
	LogicalId string
	Node      *yaml.Node
}

func (t Template) GetResourcesOfType(typeName string) []*Resource {
	resources, err := t.GetSection(Resources)
	if err != nil {
		config.Debugf("GetResourcesOfType error: %v", err)
		return nil
	}
	retval := make([]*Resource, 0)
	for i := 0; i < len(resources.Content); i += 2 {
		logicalId := resources.Content[i].Value
		resource := resources.Content[i+1]
		_, typ, _ := s11n.GetMapValue(resource, "Type")
		if typ == nil {
			continue
		}
		if typ.Value == typeName {
			retval = append(retval, &Resource{LogicalId: logicalId, Node: resource})
		}
	}
	return retval
}
