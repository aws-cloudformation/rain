package cft

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/internal/config"
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

	// If this module is wrapped in Fn::ForEach, this will be populated
	FnForEach *FnForEach
}

func (c *ModuleConfig) Properties() map[string]any {
	return node.DecodeMap(c.PropertiesNode)
}

func (c *ModuleConfig) Overrides() map[string]any {
	return node.DecodeMap(c.OverridesNode)
}

// ResourceOverridesNode returns the Overrides node for the
// given resource if it exists
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
	ForEach    string = "Fn::ForEach"
)

// parseModuleConfig parses a single module configuration
// from the Modules section in the template
func (t *Template) ParseModuleConfig(
	name string, n *yaml.Node) (*ModuleConfig, error) {

	m := &ModuleConfig{}
	m.Name = name
	m.Node = n

	// Handle Fn::ForEach modules
	if strings.HasPrefix(name, ForEach) && n.Kind == yaml.SequenceNode {
		if len(n.Content) != 3 {
			msg := "expected %s len 3, got %d"
			return nil, fmt.Errorf(msg, name, len(n.Content))
		}

		m.FnForEach = &FnForEach{}

		loopName := strings.Replace(name, ForEach, "", 1)
		loopName = strings.Replace(loopName, ":", "", -1)
		m.FnForEach.LoopName = loopName

		m.Name = loopName //  TODO: ?

		m.FnForEach.Identifier = n.Content[0].Value
		m.FnForEach.Collection = n.Content[1]
		outputKeyValue := n.Content[2]

		if outputKeyValue.Kind != yaml.MappingNode ||
			len(outputKeyValue.Content) != 2 ||
			outputKeyValue.Content[1].Kind != yaml.MappingNode {
			msg := "invalid %s, expected OutputKey: OutputValue mapping"
			return nil, fmt.Errorf(msg, name)
		}

		m.FnForEach.OutputKey = outputKeyValue.Content[0].Value
		m.Node = outputKeyValue.Content[1]
		m.FnForEach.OutputValue = m.Node
		n = m.Node
		m.Map = m.FnForEach.Collection

		config.Debugf("ModuleConfig.FnForEach: %+v", m.FnForEach)

	}

	if n.Kind != yaml.MappingNode {
		config.Debugf("ParseModuleConfig %s: %s", name, node.ToSJson(n))
		return nil, errors.New("not a mapping node")
	}

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
		case "ForEach":
			m.Map = val
		}
	}

	return m, nil
}

type FnForEach struct {
	LoopName    string
	Identifier  string
	Collection  *yaml.Node
	OutputKey   string
	OutputValue *yaml.Node
}

// OutputKeyHasIdentifier returns true if the key uses the identifier
func (ff *FnForEach) OutputKeyHasIdentifier() bool {
	dollar := "${" + ff.Identifier + "}"
	amper := "&{" + ff.Identifier + "}"
	if strings.Contains(ff.OutputKey, dollar) {
		return true
	}
	if strings.Contains(ff.OutputKey, amper) {
		return true
	}
	return false
}

// ReplaceIdentifier replaces instance of the identifier in s for collection
// key k
func ReplaceIdentifier(s, k, identifier string) string {
	dollar := "${" + identifier + "}"
	amper := "&{" + identifier + "}"
	s = strings.Replace(s, dollar, k, -1)
	s = strings.Replace(s, amper, k, -1)
	return s
}
