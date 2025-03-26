// This file implements !Rain::Module
package pkg

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	// AWS CLI package Modules compatibility.
	// This is basically the same as !Rain::Module but the modules are
	// defined in a new Modules Section.
	_, moduleSection, _ := s11n.GetMapValue(n, "Modules")
	if moduleSection == nil {
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
			parsed.Node,
			outputNode,
			t,
			parsed.AsTemplate.Constants,
			moduleConfig)
		if err != nil {
			return err
		}

		// Put the content into the template
		if len(outputNode.Content) > 0 {
			resources, err := t.GetSection(cft.Resources)
			if err != nil {
				resources = node.AddMap(t.Node.Content[0], string(cft.Resources))
			}
			resources.Content = append(resources.Content, outputNode.Content...)
		} else {
			config.Debugf("Module %s did not have any Resources", name)
		}

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
	module *yaml.Node,
	outputNode *yaml.Node,
	t *cft.Template,
	moduleConstants map[string]*yaml.Node,
	moduleConfig *cft.ModuleConfig) error {

	if moduleConfig == nil {
		return errors.New("moduleConfig is nil")
	}

	m := &Module{}
	m.Config = moduleConfig
	m.Node = module
	m.Parent = t

	// Locate the Resources: section in the module
	_, moduleResources, _ := s11n.GetMapValue(module, string(cft.Resources))
	m.ResourcesNode = moduleResources

	// Locate the Parameters: section in the module (might be nil)
	_, moduleParams, _ := s11n.GetMapValue(module, string(cft.Parameters))
	m.ParametersNode = moduleParams

	// Locate the Outputs: section in the module (might be nil)
	_, moduleOutputs, _ := s11n.GetMapValue(module, string(cft.Outputs))
	m.OutputsNode = moduleOutputs

	// Locate the Conditions: section in the module (might be nil)
	_, moduleConditions, _ := s11n.GetMapValue(module, string(cft.Conditions))
	m.ConditionsNode = moduleConditions

	// Locate the Modules: section in the module (might be nil)
	_, moduleModules, _ := s11n.GetMapValue(module, string(cft.Modules))
	m.ModulesNode = moduleModules

	err := m.ProcessConditions()
	if err != nil {
		return err
	}

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

// ProcessResources injects the module's resources into the output node
func (module *Module) ProcessResources(outputNode *yaml.Node) error {

	if module.ResourcesNode == nil {
		config.Debugf("Module %s has no resources", module.Config.Name)
		return nil
	}

	// Get module resources and add them to the output
	for i, moduleResource := range module.ResourcesNode.Content {
		if moduleResource.Kind != yaml.MappingNode {
			continue
		}
		name := module.ResourcesNode.Content[i-1].Value

		// Check to see if there is a Rain attribute in the Metadata.
		// If so, check conditionals like IfParam
		metadata := s11n.GetMap(moduleResource, Metadata)
		if metadata != nil {
			if rainMetadata, ok := metadata[Rain]; ok {
				if omitIfs(rainMetadata, module.ParametersNode, module.Config.PropertiesNode, moduleResource) {
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
	source string) error {

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

	return processModule(logicalId, module, outputNode, t, moduleConstants, moduleConfig)
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

type ModuleContent struct {
	Content    []byte
	NewRootDir string
	BaseUri    string
}

// Get the module's content from a local file, memory, or a remote uri
func getModuleContent(
	root string,
	t *cft.Template,
	templateFiles *embed.FS,
	baseUri string,
	uri string) (*ModuleContent, error) {

	var content []byte
	var err error
	var newRootDir string

	// Check to see if this is an alias like "alias/foo.yaml"
	packageAlias := checkPackageAlias(t, uri)
	isZip := false
	if packageAlias != nil {
		path := strings.Replace(uri, packageAlias.Alias+"/", "", 1)
		if strings.HasSuffix(packageAlias.Location, ".zip") {
			// Unzip, verify hash if there is one, and put the files in memory
			isZip = true
			content, err = DownloadFromZip(packageAlias.Location, packageAlias.Hash, path)
			if err != nil {
				return nil, err
			}
		} else {
			uri = strings.Replace(uri, packageAlias.Alias, packageAlias.Location, 1)
		}
	}

	// Is this a local file or a URL or did we already unzip a package?
	if isZip {
		config.Debugf("Got content from a zipped module package: %s", string(content))
	} else if strings.HasPrefix(uri, "https://") {

		content, err = downloadModule(uri)
		if err != nil {
			return nil, err
		}

		// Once we see a URL instead of a relative local path,
		// we need to remember the base URL so that we can
		// fix relative paths in any referenced modules.

		// Strip the file name from the uri
		urlParts := strings.Split(uri, "/")
		baseUri = strings.Join(urlParts[:len(urlParts)-1], "/")

	} else {
		if baseUri != "" {
			// If we have a base URL, prepend it to the relative path
			uri = baseUri + "/" + uri
			content, err = downloadModule(uri)
			if err != nil {
				return nil, err
			}
		} else if templateFiles != nil {
			// Read from the embedded file system (for the build -r command)
			// We have to hack this since embed doesn't understand "path/../"
			embeddedPath := strings.Replace(root, "../", "", 1) +
				"/" + strings.Replace(uri, "../", "", 1)

			content, err = templateFiles.ReadFile(embeddedPath)
			if err != nil {
				return nil, err
			}
			newRootDir = filepath.Dir(embeddedPath)
		} else {
			// Read the local file
			path := uri
			if !filepath.IsAbs(path) {
				path = filepath.Join(root, path)
			}

			info, err := os.Stat(path)
			if err != nil {
				return nil, err
			}

			if info.IsDir() {
				return nil, fmt.Errorf("'%s' is a directory", path)
			}

			content, err = os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			newRootDir = filepath.Dir(path)
		}
	}

	return &ModuleContent{content, newRootDir, baseUri}, nil
}

type ParsedModule struct {
	Node       *yaml.Node
	AsTemplate *cft.Template
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

	// Treat the module as a template
	moduleAsTemplate := cft.Template{Node: &moduleNode}

	// Read things like Constants, Modules, Packages
	processRainSection(&moduleAsTemplate, rootDir, fs)
	processAddedSections(&moduleAsTemplate, moduleAsTemplate.Node.Content[0], rootDir, fs)

	return &ParsedModule{Node: moduleNode.Content[0], AsTemplate: &moduleAsTemplate}, nil
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
	err = processRainResourceModule(moduleNode, &outputNode, t, parent, moduleAsTemplate.Constants, uri)
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
