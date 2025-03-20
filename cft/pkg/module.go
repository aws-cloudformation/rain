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
	"github.com/aws-cloudformation/rain/cft/visitor"
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
)

// Module represents a complete module, including parent config
type Module struct {
	Config         *ModuleConfig
	ParametersNode *yaml.Node
	ResourcesNode  *yaml.Node
	OutputsNode    *yaml.Node
	Node           *yaml.Node
	Parent         *cft.Template
}

// Outputs returns the Outputs node as a map, which is easier to use
func (module *Module) Outputs() map[string]any {
	return decodeMapNode(module.OutputsNode)
}

// Parameters returns the Parameters node as a map, which is easier to use
func (module *Module) Parameters() map[string]any {
	return decodeMapNode(module.ParametersNode)
}

// Resources returns the Resources node as a map, which is easier to use
func (module *Module) Resources() map[string]any {
	return decodeMapNode(module.ResourcesNode)
}

func decodeMapNode(n *yaml.Node) map[string]any {
	var m map[string]any
	if n != nil {
		decodeErr := n.Decode(&m)
		if decodeErr != nil {
			config.Debugf("decodeMapNode error: %v", decodeErr)
			return m
		}
	}
	return m
}

// ModuleConfig is the configuration of the module in
// the parent template.
type ModuleConfig struct {

	// Name is the name of the module, which is used as a logical id prefix
	Name string

	// Source is the URI for the module, a local file or remote URL
	Source string

	// Properties map to the module's Parameters
	Properties map[string]any

	// PropertiesNode is the yaml node for the Properties
	PropertiesNode *yaml.Node

	// Overrides are used to directly override module content
	Overrides map[string]any

	// OverridesNode is the yaml node for overrides
	OverridesNode *yaml.Node

	// Node is the yaml node for the Module config mapping node
	Node *yaml.Node

	// Map is the node for mapping (duplicating) the module based on a CSV
	Map *yaml.Node

	// OriginalName is the Name of the module before it got Mapped (duplicated)
	OriginalName string
}

// parseModuleConfig parses a single module configuration
// from the Modules section in the template
func parseModuleConfig(name string, n *yaml.Node) (*ModuleConfig, error) {
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
		case "Source":
			m.Source = val.Value
		case "Properties":
			m.PropertiesNode = val
			decodeErr := val.Decode(&m.Properties)
			if decodeErr != nil {
				return nil, decodeErr
			}
		case "Overrides":
			m.OverridesNode = val
			decodeErr := val.Decode(&m.Overrides)
			if decodeErr != nil {
				return nil, decodeErr
			}
		case "Map":
			m.Map = val
		}
	}

	return m, nil
}

// stringsFromNode returns a string slice based on a scalar with a CSV or a sequence.
func stringsFromNode(n *yaml.Node) []string {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.ScalarNode {
		return strings.Split(n.Value, ",")
	}
	if n.Kind == yaml.SequenceNode {
		retval := make([]string, 0)
		for _, s := range n.Content {
			retval = append(retval, s.Value)
		}
		return retval
	}
	return nil
}

const (
	MI = "$MapIndex"
	MV = "$MapValue"
)

func replaceMapStr(s string, index int, key string) string {
	s = strings.Replace(s, MI, fmt.Sprintf("%d", index), -1)
	s = strings.Replace(s, MV, key, -1)
	return s
}

// mapPlaceholders looks for MapValue and MapIndex and replaces them
func mapPlaceholders(n *yaml.Node, index int, key string) {

	vf := func(v *visitor.Visitor) {
		yamlNode := v.GetYamlNode()
		if yamlNode.Kind == yaml.MappingNode {
			content := yamlNode.Content
			if len(content) == 2 {
				switch content[0].Value {
				case string(cft.Sub):
					r := replaceMapStr(content[1].Value, index, key)
					if parse.IsSubNeeded(r) {
						yamlNode.Value = r
					} else {
						*yamlNode = *node.MakeScalarNode(r)
					}
				case string(cft.GetAtt):
					for _, getatt := range content[1].Content {
						getatt.Value = replaceMapStr(getatt.Value, index, key)
					}
				}
			}
		} else if yamlNode.Kind == yaml.ScalarNode {
			yamlNode.Value = replaceMapStr(yamlNode.Value, index, key)
		}
	}

	visitor.NewVisitor(n).Visit(vf)
}

// Process the Modules section of a template or module.
// Modifies t in place.
func processModulesSection(t *cft.Template, n *yaml.Node, rootDir string, fs *embed.FS) error {

	// AWS CLI package Modules compatibility.
	// This is basically the same as !Rain::Module but the modules are
	// defined in a new Modules Section.
	_, moduleSection, _ := s11n.GetMapValue(n, "Modules")
	if moduleSection == nil {
		config.Debugf("No modules section")
		return nil
	}
	config.Debugf("Modules:\n%v", moduleSection)
	HasModules = true

	if moduleSection.Kind != yaml.MappingNode {
		return errors.New("the Modules section is not a mapping node")
	}

	originalContent := moduleSection.Content

	//config.Debugf("originalContent: \n%s", node.ToSJson(moduleSection))

	content := make([]*yaml.Node, 0)

	// This will hold info about original mapped modules
	moduleMaps := make(map[string]any)

	// Process Maps, which duplicate the module for each element in a list
	for i := 0; i < len(originalContent); i += 2 {
		name := originalContent[i].Value
		moduleConfig, err := parseModuleConfig(name, originalContent[i+1])
		if err != nil {
			return err
		}

		//config.Debugf("processing Maps, module %s: %+v", name, moduleConfig)

		if moduleConfig.Map != nil {
			// The map is either a CSV or a Ref to a CSV that we can fully resolve
			mapJson := node.ToSJson(moduleConfig.Map)
			var keys []string
			if moduleConfig.Map.Kind == yaml.ScalarNode {
				keys = stringsFromNode(moduleConfig.Map)
			} else if moduleConfig.Map.Kind == yaml.SequenceNode {
				keys = stringsFromNode(moduleConfig.Map)
			} else if moduleConfig.Map.Kind == yaml.MappingNode {
				if len(moduleConfig.Map.Content) == 2 && moduleConfig.Map.Content[0].Value == "Ref" {
					r := moduleConfig.Map.Content[1].Value
					// Look in the parent templates Parameters for the Ref
					if !t.HasSection(cft.Parameters) {
						return fmt.Errorf("module Map Ref no Parameters: %s", mapJson)
					}
					params, _ := t.GetSection(cft.Parameters)
					_, keysNode, _ := s11n.GetMapValue(params, r)
					if keysNode == nil {
						return fmt.Errorf("expected module Map Ref to a Parameter: %s", mapJson)
					}
					// TODO - This is too simple. What about sub-modules?
					_, d, _ := s11n.GetMapValue(keysNode, "Default")
					if d == nil {
						return fmt.Errorf("expected module Map Ref to a Default: %s", mapJson)
					}
					keys = stringsFromNode(d)
				} else {
					return fmt.Errorf("expected module Map to be a Ref: %s", mapJson)
				}
			} else {
				return fmt.Errorf("unexpected module Map Kind: %s", mapJson)
			}

			if len(keys) < 1 {
				mapErr := fmt.Errorf("expected module Map to have items: %s", mapJson)
				return mapErr
			}

			// Record the number of items in the map
			moduleMaps[moduleConfig.Name] = len(keys)

			// Duplicate the config
			for i, key := range keys {
				mapName := fmt.Sprintf("%s%d", moduleConfig.Name, i)
				copiedNode := node.Clone(moduleConfig.Node)
				node.RemoveFromMap(copiedNode, "Map")
				copiedConfig, err := parseModuleConfig(mapName, copiedNode)
				if err != nil {
					return err
				}
				copiedConfig.OriginalName = moduleConfig.Name
				mapPlaceholders(copiedConfig.Node, i, key)
				content = append(content, node.MakeScalarNode(mapName))
				content = append(content, copiedNode)
			}
		} else {
			content = append(content, originalContent[i])
			content = append(content, originalContent[i+1])
		}
	}

	//config.Debugf("content after Maps: \n%s",
	//	node.ToSJson(&yaml.Node{Kind: yaml.MappingNode, Content: content}))

	for i := 0; i < len(content); i += 2 {
		name := content[i].Value
		moduleConfig, err := parseModuleConfig(name, content[i+1])
		if err != nil {
			return err
		}
		//config.Debugf("Module Config: %+v", moduleConfig)

		baseUri := ""
		uri := moduleConfig.Source

		moduleContent, err := getModuleContent(rootDir, t, fs, baseUri, uri)
		if err != nil {
			return err
		}
		//config.Debugf("Module %s content:\n%s", name, moduleContent.Content)

		parsed, err := parseModule(moduleContent.Content, rootDir, fs)
		if err != nil {
			return err
		}

		//config.Debugf("Module %s Parsed: %s", name, node.ToSJson(parsed.Node))

		// Transform the parsed module content
		outputNode := node.MakeMappingNode()

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

		//config.Debugf("outputNode:\n%s", node.ToSJson(outputNode))

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

	// Remove the Modules section
	t.RemoveSection(cft.Modules)

	return nil
}

// Clone a property-like node from the module and replace any overridden values
func cloneAndReplaceProps(
	moduleProps *yaml.Node,
	templateProps *yaml.Node,
	moduleParams *yaml.Node) *yaml.Node {

	// Not all property-like attributes are required
	if moduleProps == nil && templateProps == nil {
		return nil
	}

	var props *yaml.Node

	if moduleProps != nil {
		// Start by cloning the properties in the module
		props = node.Clone(moduleProps)
	} else {
		props = &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
	}

	// Replace any property values overridden in the parent template
	if templateProps != nil {
		for i, tprop := range templateProps.Content {

			// Only look at the names, which have even indexes
			if i%2 != 0 {
				continue
			}

			found := false

			if moduleParams != nil {
				_, moduleParam, _ := s11n.GetMapValue(moduleParams, tprop.Value)

				// Don't clone template props that are module parameters.
				// Module params are used when we resolve Refs later
				if moduleParam != nil {
					continue
				}
			}

			// Overwrite anything hard coded into the module that is
			// present in the parent template
			for j, mprop := range props.Content {
				if tprop.Value == mprop.Value && i%2 == 0 && j%2 == 0 {
					clonedNode := node.Clone(templateProps.Content[i+1])
					// config.Debugf("original: %s", node.ToSJson(templateProps.Content[i+1]))
					// config.Debugf("clonedNode: %s", node.ToSJson(clonedNode))
					merged := node.MergeNodes(props.Content[j+1], clonedNode)
					// config.Debugf("merged: %s", node.ToSJson(merged))
					props.Content[j+1] = merged

					found = true
				}
			}

			if !found && i%2 == 0 {
				props.Content = append(props.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: tprop.Value})
				props.Content = append(props.Content, node.Clone(templateProps.Content[i+1]))
			}

		}
	}

	return props
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
	moduleConfig *ModuleConfig) error {

	m := &Module{}
	m.Config = moduleConfig
	m.Node = module
	m.Parent = t

	// Locate the Resources: section in the module
	_, moduleResources, _ := s11n.GetMapValue(module, "Resources")
	m.ResourcesNode = moduleResources

	// Locate the Parameters: section in the module (might be nil)
	_, moduleParams, _ := s11n.GetMapValue(module, "Parameters")
	m.ParametersNode = moduleParams

	// Locate the Outputs: section in the module (might be nil)
	_, moduleOutputs, _ := s11n.GetMapValue(module, "Outputs")
	m.OutputsNode = moduleOutputs

	err := validateOverrides(moduleConfig.Node, moduleResources, moduleParams)
	if err != nil {
		return err
	}

	// Get module resources and add them to the output
	for i, moduleResource := range moduleResources.Content {
		if moduleResource.Kind != yaml.MappingNode {
			continue
		}
		name := moduleResources.Content[i-1].Value

		// Check to see if there is a Rain attribute in the Metadata.
		// If so, check conditionals like IfParam
		metadata := s11n.GetMap(moduleResource, Metadata)
		if metadata != nil {
			if rainMetadata, ok := metadata[Rain]; ok {
				if omitIfs(rainMetadata, moduleParams, moduleConfig.PropertiesNode, moduleResource) {
					continue
				}
			}
		}

		nameNode := node.Clone(moduleResources.Content[i-1])
		nameNode.Value = rename(logicalId, nameNode.Value)
		outputNode.Content = append(outputNode.Content, nameNode)
		clonedResource := node.Clone(moduleResource)

		_, err := processOverrides(logicalId, moduleConfig.Node, name, moduleResource, clonedResource, moduleParams)
		if err != nil {
			return err
		}

		// Resolve Refs in the module
		// Some refs are to other resources in the module
		// Other refs are to the module's parameters
		err = m.Resolve(clonedResource)
		if err != nil {
			return fmt.Errorf("failed to resolve refs: %v", err)
		}

		outputNode.Content = append(outputNode.Content, clonedResource)
	}

	// Look for references to this module's outputs in the parent
	err = m.ProcessOutputs()
	if err != nil {
		return err
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

	moduleConfig, err := parseModuleConfig(logicalId, templateResource)
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
		config.Debugf("Found package alias: %+v", packageAlias)
		path := strings.Replace(uri, packageAlias.Alias+"/", "", 1)
		config.Debugf("path is %s", path)
		if strings.HasSuffix(packageAlias.Location, ".zip") {
			// Unzip, verify hash if there is one, and put the files in memory
			isZip = true
			content, err = DownloadFromZip(packageAlias.Location, packageAlias.Hash, path)
			if err != nil {
				return nil, err
			}
		} else {
			uri = strings.Replace(uri, packageAlias.Alias, packageAlias.Location, 1)
			config.Debugf("uri is now %s", uri)
			config.Debugf("baseUri is %s", baseUri)
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
