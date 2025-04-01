package cft

import (
	"errors"

	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// ModuleConfig is the configuration of the module in the parent template.
type ModuleConfig struct {

	// Name is the name of the module, which is used as a logical id prefix
	Name string

	// Source is the URI for the module, a local file or remote URL
	Source string

	// PropertiesNode is the yaml node for the Properties
	PropertiesNode *yaml.Node

	// OverridesNode is the yaml node for overrides
	OverridesNode *yaml.Node

	// Node is the yaml node for the Module config mapping node
	Node *yaml.Node

	// Map is the node for mapping (duplicating) the module based on a CSV
	Map *yaml.Node

	// OriginalName is the Name of the module before it got Mapped (duplicated)
	OriginalName string

	// If this module was duplicated because of the Map attribute, store the index
	MapIndex int

	// If this module was duplicated because of the Map attribute, store the key
	MapKey string

	// IsMapCopy will be true if this instance was a duplicate of a Mapped module
	IsMapCopy bool

	// The root directory of the template that configures this module
	ParentRootDir string
}

func (c *ModuleConfig) Properties() map[string]any {
	return node.DecodeMap(c.PropertiesNode)
}

func (c *ModuleConfig) Overrides() map[string]any {
	return node.DecodeMap(c.OverridesNode)
}

// ResourceOverridesNode returns the Overrides node for the given resource if it exists
func (c *ModuleConfig) ResourceOverridesNode(name string) *yaml.Node {
	if c.OverridesNode == nil {
		return nil
	}
	_, n, _ := s11n.GetMapValue(c.OverridesNode, name)
	return n
}

const (
	Source     string = "Source"
	Properties string = "Properties"
	Overrides  string = "Overrides"
	Map        string = "Map"
)

// parseModuleConfig parses a single module configuration
// from the Modules section in the template
func ParseModuleConfig(name string, n *yaml.Node) (*ModuleConfig, error) {
	if n.Kind != yaml.MappingNode {
		return nil, errors.New("not a mapping node")
	}
	m := &ModuleConfig{}
	m.Name = name
	m.Node = n

	content := n.Content
	for i := 0; i < len(content); i += 2 {
		attr := content[i].Value
		val := content[i+1]
		switch attr {
		case Source:
			m.Source = val.Value
		case Properties:
			m.PropertiesNode = val
		case Overrides:
			m.OverridesNode = val
		case Map:
			m.Map = val
		}
	}

	return m, nil
}
