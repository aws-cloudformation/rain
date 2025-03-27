// This file implements !Rain::Module
package pkg

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

var HasModules bool

const (
	Metadata            = "Metadata"
	IfParam             = "IfParam"
	IfNotParam          = "IfNotParam"
	Overrides           = "Overrides"
	DependsOn           = "DependsOn"
	Properties          = "Properties"
	CreationPolicy      = "CreationPolicy"
	UpdatePolicy        = "UpdatePolicy"
	DeletionPolicy      = "DeletionPolicy"
	UpdateReplacePolicy = "UpdateReplacePolicy"
	Condition           = "Condition"
	Default             = "Default"
	Source              = "Source"
	Map                 = "Map"
)

// Module represents a complete module, including parent config
type Module struct {
	Config         *cft.ModuleConfig
	ParametersNode *yaml.Node
	ResourcesNode  *yaml.Node
	OutputsNode    *yaml.Node
	Node           *yaml.Node
	Parent         *cft.Template
	ConditionsNode *yaml.Node
	ModulesNode    *yaml.Node
}

// Outputs returns the Outputs node as a map
func (module *Module) Outputs() map[string]any {
	return node.DecodeMap(module.OutputsNode)
}

// Parameters returns the Parameters node as a map
func (module *Module) Parameters() map[string]any {
	return node.DecodeMap(module.ParametersNode)
}

// Resources returns the Resources node as a map
func (module *Module) Resources() map[string]any {
	return node.DecodeMap(module.ResourcesNode)
}

// Modules returns the Modules node as a map
func (module *Module) Modules() map[string]any {
	return node.DecodeMap(module.ModulesNode)
}

// Conditions returns the Conditions node as a map
func (module *Module) Conditions() map[string]any {
	return node.DecodeMap(module.ConditionsNode)
}

// Process the Modules section of a template or module.
// Modifies t in place.
func processModulesSection(t *cft.Template, n *yaml.Node, rootDir string, fs *embed.FS) error {

	config.Debugf("processModulesSection t:\n%s", node.YamlStr(t.Node))
	config.Debugf("processModulesSection n:\n%s", node.ToSJson(n))

	// AWS CLI package Modules compatibility.
	// This is basically the same as !Rain::Module but the modules are
	// defined in a new Modules Section.
	_, moduleSection, _ := s11n.GetMapValue(n, "Modules")
	if moduleSection == nil {
		config.Debugf("No Modules section found")
		return nil
	}
	HasModules = true

	if moduleSection.Kind != yaml.MappingNode {
		return errors.New("the Modules section is not a mapping node")
	}

	originalContent := moduleSection.Content

	// Duplicate module content that has a Map attribute
	content, err := processMaps(originalContent, t)
	if err != nil {
		return err
	}

	// Replace the original Modules content
	moduleSection.Content = content

	config.Debugf("content after Maps: \n%s", node.YamlStr(moduleSection))

	for i := 0; i < len(content); i += 2 {
		name := content[i].Value
		moduleConfig, err := cft.ParseModuleConfig(name, content[i+1])
		if err != nil {
			return err
		}

		baseUri := ""
		uri := moduleConfig.Source

		moduleContent, err := getModuleContent(rootDir, t, fs, baseUri, uri)
		if err != nil {
			return err
		}

		parsed, err := parseModule(moduleContent.Content, rootDir, fs)
		if err != nil {
			return err
		}

		// Transform the parsed module content
		outputNode := node.MakeMapping()

		err = processModule(
			name,
			parsed,
			outputNode,
			t,
			parsed.AsTemplate.Constants,
			moduleConfig)
		if err != nil {
			return err
		}

		config.Debugf("after processModule %s outputNode:\n%s",
			name, node.ToSJson(outputNode))

		config.Debugf("t before adding resources:\n%s", node.ToSJson(t.Node))

		// Put the content into the template
		if len(outputNode.Content) > 0 {
			resources, err := t.GetSection(cft.Resources)
			if err != nil {
				resources = node.AddMap(t.Node.Content[0], string(cft.Resources))
			}
			resources.Content = append(resources.Content, outputNode.Content...)

			config.Debugf("resources after append:\n%s", node.YamlStr(resources))
		} else {
			config.Debugf("processModuleSection %s outputNode did not have any Resources", name)
		}

		config.Debugf("t after processModule:\n%s", node.ToSJson(t.Node))

	}

	// Look for GetAtts like Content[].Arn that reference
	// all items in Mapped module Outputs
	ProcessOutputArrays(t)

	// Remove the Modules section
	t.RemoveSection(cft.Modules)

	return nil
}

// Add DeletionPolicy, UpdateReplacePolicy, and Condition
func addScalarAttribute(out *yaml.Node, name string, moduleResource *yaml.Node, overrides *yaml.Node) {
	_, templatePolicy, _ := s11n.GetMapValue(overrides, name)
	_, modulePolicy, _ := s11n.GetMapValue(moduleResource, name)
	if modulePolicy != nil {
		node.RemoveFromMap(out, name)
	}
	if templatePolicy != nil || modulePolicy != nil {
		policy := &yaml.Node{Kind: yaml.ScalarNode, Value: name}
		var policyValue *yaml.Node
		if templatePolicy != nil {
			policyValue = node.Clone(templatePolicy)
		} else {
			policyValue = node.Clone(modulePolicy)
		}
		out.Content = append(out.Content, policy)
		out.Content = append(out.Content, policyValue)
	}
}

// processModule performs all of the module logic and injects the content into the parent
func processModule(
	logicalId string,
	parsedModule *ParsedModule,
	outputNode *yaml.Node,
	t *cft.Template,
	moduleConstants map[string]*yaml.Node,
	moduleConfig *cft.ModuleConfig) error {

	if moduleConfig == nil {
		return errors.New("moduleConfig is nil")
	}

	moduleNode := parsedModule.Node

	config.Debugf("processModule %s:\n%s", logicalId, node.YamlStr(moduleNode))

	m := &Module{}
	m.Config = moduleConfig
	m.Node = moduleNode
	m.Parent = t

	m.InitNodes()

	err := m.ProcessConditions()
	if err != nil {
		return err
	}

	moduleAsTemplate := &cft.Template{
		Node: &yaml.Node{
			Kind:    yaml.DocumentNode,
			Content: []*yaml.Node{moduleNode},
		},
	}

	// Move processRainSection and processAddedSections here?
	processRainSection(moduleAsTemplate,
		parsedModule.RootDir, parsedModule.FS)

	processAddedSections(moduleAsTemplate, moduleAsTemplate.Node.Content[0],
		parsedModule.RootDir, parsedModule.FS)

	err = m.ValidateOverrides()
	if err != nil {
		return err
	}

	err = m.ProcessResources(outputNode)
	if err != nil {
		return err
	}

	// Look for references to this module's outputs in the parent
	err = m.ProcessOutputs()
	if err != nil {
		return err
	}

	// Resolve any references to this module in the parent template
	err = m.Resolve(t.Node)
	if err != nil {
		return err
	}

	return nil
}

func (module *Module) InitNodes() {
	// Locate the Resources: section in the module
	_, moduleResources, _ := s11n.GetMapValue(module.Node, string(cft.Resources))
	module.ResourcesNode = moduleResources

	// Locate the Parameters: section in the module (might be nil)
	_, moduleParams, _ := s11n.GetMapValue(module.Node, string(cft.Parameters))
	module.ParametersNode = moduleParams

	// Locate the Outputs: section in the module (might be nil)
	_, moduleOutputs, _ := s11n.GetMapValue(module.Node, string(cft.Outputs))
	module.OutputsNode = moduleOutputs

	// Locate the Conditions: section in the module (might be nil)
	_, moduleConditions, _ := s11n.GetMapValue(module.Node, string(cft.Conditions))
	module.ConditionsNode = moduleConditions

	// Locate the Modules: section in the module (might be nil)
	_, moduleModules, _ := s11n.GetMapValue(module.Node, string(cft.Modules))
	module.ModulesNode = moduleModules
}

// ProcessResources injects the module's resources into the output node
func (module *Module) ProcessResources(outputNode *yaml.Node) error {

	config.Debugf("ProcessResources %s: %s",
		module.Config.Name, node.ToSJson(module.Node))

	// Resources Node may have been replaced
	module.InitNodes()

	if module.ResourcesNode == nil {
		config.Debugf("Module %s has no resources", module.Config.Name)
		return nil
	}

	config.Debugf("ProcessResources %s has %d Resources", module.Config.Name,
		len(module.ResourcesNode.Content)/2)

	// Get module resources and add them to the output
	for i, moduleResource := range module.ResourcesNode.Content {
		if moduleResource.Kind != yaml.MappingNode {
			continue
		}
		name := module.ResourcesNode.Content[i-1].Value

		config.Debugf("ProcessResources %s, resource %s",
			module.Config.Name, name)

		// Check to see if there is a Rain attribute in the Metadata.
		// If so, check conditionals like IfParam
		metadata := s11n.GetMap(moduleResource, Metadata)
		if metadata != nil {
			if rainMetadata, ok := metadata[Rain]; ok {
				if omitIfs(rainMetadata, module.ParametersNode,
					module.Config.PropertiesNode, moduleResource) {
					continue
				}
			}
		}

		nameNode := node.Clone(module.ResourcesNode.Content[i-1])
		nameNode.Value = rename(module.Config.Name, nameNode.Value)
		outputNode.Content = append(outputNode.Content, nameNode)
		clonedResource := node.Clone(moduleResource)

		err := module.ProcessOverrides(name, moduleResource, clonedResource)
		if err != nil {
			return err
		}

		// Resolve Refs in the module
		// Some refs are to other resources in the module
		// Other refs are to the module's parameters
		err = module.Resolve(clonedResource)
		if err != nil {
			return fmt.Errorf("failed to resolve refs: %v", err)
		}

		outputNode.Content = append(outputNode.Content, clonedResource)
	}

	config.Debugf("ProcessResources outputNode: %s", node.ToSJson(outputNode))

	return nil
}

// Convert the module into a node for the packaged template
// This is for !Rain::Module Resources
func processRainResourceModule(
	module *yaml.Node,
	outputNode *yaml.Node,
	t *cft.Template,
	parent node.NodePair,
	moduleConstants map[string]*yaml.Node,
	source string,
	parsed *ParsedModule) error {

	// The parent arg is the map in the template resource's Content[1] that contains Type, Properties, etc

	if parent.Key == nil {
		return errors.New("expected parent.Key to not be nil. The !Rain::Module directive should come after Type: ")
	}

	// Get the logical id of the resource we are transforming
	logicalId := parent.Key.Value

	// Make a new node that will hold our additions to the original template
	outputNode.Content = make([]*yaml.Node, 0)

	if module.Kind == yaml.DocumentNode {
		module = module.Content[0] // ScalarNode !!map
	}

	if module.Kind != yaml.MappingNode {
		config.Debugf("%s", node.ToSJson(module))
		return fmt.Errorf("expected module %s to be a Mapping node", logicalId)
	}

	templateResource := parent.Value // The !!map node of the resource with Type !Rain::Module

	moduleConfig, err := cft.ParseModuleConfig(logicalId, templateResource)
	if err != nil {
		return err
	}
	moduleConfig.Source = source

	return processModule(logicalId, parsed, outputNode, t, moduleConstants, moduleConfig)
}

func checkPackageAlias(t *cft.Template, uri string) *cft.PackageAlias {
	tokens := strings.Split(uri, "/")
	if len(tokens) > 1 {
		// See if this is one of the template package aliases
		if t.Packages != nil {
			if p, ok := t.Packages[tokens[0]]; ok {
				return p
			}
		}
	}
	return nil
}

type ParsedModule struct {
	Node       *yaml.Node
	AsTemplate *cft.Template
	RootDir    string
	FS         *embed.FS
}

// parseModule parses module content and converts it to a yaml node
// Also process new sections: Rain, Constants, Modules, Packages
func parseModule(content []byte, rootDir string, fs *embed.FS) (*ParsedModule, error) {

	var err error

	// Parse the file
	var moduleNode yaml.Node
	err = yaml.Unmarshal(content, &moduleNode)
	if err != nil {
		return nil, err
	}

	err = parse.NormalizeNode(&moduleNode)
	if err != nil {
		return nil, err
	}

	config.Debugf("parseModule: \n%s", node.ToSJson(&moduleNode))

	// Treat the module as a template
	moduleAsTemplate := cft.Template{Node: &moduleNode}

	// Read things like Constants, Modules, Packages
	//processRainSection(&moduleAsTemplate, rootDir, fs)
	//processAddedSections(&moduleAsTemplate, moduleAsTemplate.Node.Content[0], rootDir, fs)
	// TODO: Move these out for later?

	return &ParsedModule{
		Node:       moduleNode.Content[0],
		AsTemplate: &moduleAsTemplate,
		RootDir:    rootDir,
		FS:         fs,
	}, nil
}

// Type: !Rain::Module
// This handles the Rain Module directive, not the Modules section
func module(ctx *directiveContext) (bool, error) {

	n := ctx.n
	t := ctx.t
	parent := ctx.parent

	if !Experimental {
		panic("You must add the --experimental arg to use the !Rain::Module directive")
	}

	if len(n.Content) != 2 {
		return false, errors.New("expected !Rain::Module <URI>")
	}

	HasModules = true

	uri := n.Content[1].Value

	moduleContent, err := getModuleContent(ctx.rootDir,
		ctx.t, ctx.fs, ctx.baseUri, uri)
	if err != nil {
		return false, err
	}

	content := moduleContent.Content
	baseUri := moduleContent.BaseUri

	parsed, err := parseModule(content, ctx.rootDir, ctx.fs)
	if err != nil {
		return false, err
	}
	moduleNode := parsed.Node
	moduleAsTemplate := parsed.AsTemplate

	// Figure out parent nodes to handle nested modules
	var newParent node.NodePair
	if parent.Parent != nil && parent.Parent.Value != nil {
		newParent = node.GetParent(n, parent.Parent.Value, nil)
		newParent.Parent = &parent
	}

	_, err = transform(&transformContext{
		nodeToTransform: moduleNode,
		rootDir:         moduleContent.NewRootDir,
		t:               moduleAsTemplate,
		parent:          &newParent,
		fs:              ctx.fs,
		baseUri:         baseUri,
	})
	if err != nil {
		return false, err
	}

	// Create a new node to represent the parsed module
	var outputNode yaml.Node
	err = processRainResourceModule(moduleNode,
		&outputNode, t, parent, moduleAsTemplate.Constants, uri, parsed)
	if err != nil {
		config.Debugf("processModule error: %v, moduleNode: %s", err, node.ToSJson(moduleNode))
		return false, fmt.Errorf("failed to process module %s: %v", uri, err)
	}

	// Find the resource node in the template
	_, resourceNode, _ := s11n.GetMapValue(t.Node.Content[0], "Resources")
	if resourceNode == nil {
		return false, errors.New("expected template to have Resources")
	}

	// Remove the original from the template
	err = node.RemoveFromMap(resourceNode, parent.Key.Value)
	if err != nil {
		config.Debugf("err removing original: %s\n%v",
			parent.Key.Value, node.ToSJson(resourceNode))
		return false, fmt.Errorf("can't remove original from template: %v", err)
	}

	// Insert the transformed resource into the template
	resourceNode.Content = append(resourceNode.Content, outputNode.Content...)

	return true, nil

}

// processAddedSections can be used for Constants and Modules
// in either the Rain section (backwards comapatibility) or
// if they are at the top level (like the AWS CLI)
func processAddedSections(
	t *cft.Template, n *yaml.Node, rootDir string, fs *embed.FS) error {

	var err error

	err = processConstants(t, n)
	if err != nil {
		return err
	}
	err = processPackages(t, n)
	if err != nil {
		return err
	}

	err = processModulesSection(t, n, rootDir, fs)
	if err != nil {
		return err
	}

	err = FnJoin(t.Node)
	if err != nil {
		return err
	}

	return nil
}
