// Package pkg provides functionality similar to the AWS CLI cloudformation package command
// but has greater flexibility, allowing content to be included anywhere in a template
//
// To include content into your templates, use any of the following either as YAML tags
// or as one-property objects, much as AWS intrinsic functions are used, e.g. "Fn::Join"
//
// `Rain::Include`: insert the content of the file into the template directly. The file must be in YAML or JSON format.
// `Rain::Env`: inserts environmental variable value into the template as a string. Variable must be set.
// `Rain::Embed`: insert the content of the file as a string
// `Rain::S3Http`: uploads the file or directory (zipping it first) to S3 and returns the HTTP URI (i.e. `https://bucket.s3.region.amazonaws.com/key`)
// `Rain::S3`: a string value uploads the file or directory (zipping it first) to S3 and returns the S3 URI (i.e. `s3://bucket/key`)
// `Rain::S3`: an object with the following properties
//
//	`Path`: path to the file or directory. If a directory is supplied, it will be zipped before uploading to S3
//	`BucketProperty`: Name of returned property that will contain the bucket name
//	`KeyProperty`: Name of returned property that will contain the object key
//	`VersionProperty`: (optional) Name of returned property that will contain the object version
//
// `Rain::Module`: Supply a URL to a rain module, which is similar to a CloudFormation module,
//
//	but allows for type inheritance. One of the resources in the module yaml file
//	must be called "ModuleExtension", and it must have a Metadata entry called
//	"Extends" that supplies the existing type to be extended. The Parameters section
//	of the module can be used to define additional properties for the extension.
package pkg

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	rainpkl "github.com/aws-cloudformation/rain/pkl"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/cft/visitor"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// Experimental must be set to true to enable !Rain::Module
var Experimental bool
var NoAnalytics bool
var HasRainSection bool

type analytics struct {
	// Current Rain version
	Version string

	// Are we using experimental features?
	Experimental bool

	// Did we use any Rain modules?
	HasModules bool

	// Did the template have a Rain section?
	HasRainSection bool
}

type transformContext struct {
	nodeToTransform *yaml.Node
	rootDir         string // Using normal files
	t               cft.Template
	parent          *node.NodePair
	fs              *embed.FS // Used by build with embedded filesystem

	// baseUri is the base path for downloading submodules
	baseUri string
}

func transform(ctx *transformContext) (bool, error) {
	changed := false

	// config.Debugf("transform: %s", node.ToSJson(ctx.nodeToTransform))

	// registry is a map of functions defined in rain.go
	for path, fn := range registry {
		for found := range s11n.MatchAll(ctx.nodeToTransform, path) {
			nodeParent := node.GetParent(found, ctx.nodeToTransform, nil)
			nodeParent.Parent = ctx.parent
			c, err := fn(&directiveContext{found, ctx.rootDir, ctx.t, nodeParent, ctx.fs, ctx.baseUri})
			if err != nil {
				config.Debugf("Error packaging template: %s\n", err)
				return false, err
			}

			changed = changed || c
		}
	}

	return changed, nil
}

// processRainSection returns true if the Rain section of the template existed
// The section is removed by this function
func processRainSection(t *cft.Template) bool {
	t.Constants = make(map[string]*yaml.Node)
	rainNode, err := t.GetSection(cft.Rain)
	if err != nil {
		// This is okay, not all templates have a Rain section
		return false
	}

	// Process constants in order, since they can refer to previous ones
	_, c, _ := s11n.GetMapValue(rainNode, "Constants")
	if c != nil {
		for i := 0; i < len(c.Content); i += 2 {
			name := c.Content[i].Value
			val := c.Content[i+1]
			t.Constants[name] = val
			// Visit each node in val looking for any ${Rain::ConstantName}
			// and replace it with prior constant entries
			vf := func(v *visitor.Visitor) {
				vnode := v.GetYamlNode()
				if vnode.Kind == yaml.ScalarNode {
					err := replaceConstants(vnode, t.Constants)
					if err != nil {
						// These constants must be scalars
						config.Debugf("replaceConstants failed: %v", err)
					}
				}
			}
			v := visitor.NewVisitor(val)
			v.Visit(vf)
		}
	}

	// Package aliases
	packages := s11n.GetMap(rainNode, "Packages")
	if packages != nil {
		t.Packages = make(map[string]*cft.PackageAlias)
		for k, v := range packages {
			p := &cft.PackageAlias{}
			p.Alias = k
			p.Location = s11n.GetValue(v, "Location")
			p.Hash = s11n.GetValue(v, "Hash")
			t.Packages[k] = p
		}

		// Visit all resources to look for Type nodes that use $alias.module shorthand
		resources, err := t.GetSection(cft.Resources)
		if err == nil {
			for i := 0; i < len(resources.Content); i += 2 {
				resource := resources.Content[i+1]
				_, typ, _ := s11n.GetMapValue(resource, "Type")
				if typ == nil {
					continue
				}
				if strings.HasPrefix(typ.Value, "$") {
					// This is a package alias, fix it so it gets processed later
					//typ.Value = "!Rain::Module " + strings.Trim(typ.Value, "$")
					newTypeNode := yaml.Node{Kind: yaml.MappingNode}
					newTypeNode.Content = []*yaml.Node{
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Rain::Module"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: strings.Trim(typ.Value, "$")},
					}
					*typ = newTypeNode
				}
			}
		}

	}

	// Add handling for any other features we add to the Rain section here

	// Now remove the Rain node from the template
	t.RemoveSection(cft.Rain)

	HasRainSection = true

	return true
}

// Template returns t with assets included as per AWS CLI packaging rules
// and any Rain:: functions used.
// rootDir must be passed in so that any included assets can be loaded from the same directory
func Template(t cft.Template, rootDir string, fs *embed.FS) (cft.Template, error) {
	templateNode := t.Node
	var err error

	//config.Debugf("Original template short: %v", node.ToSJson(t.Node))
	//config.Debugf("Original template long: %v", node.ToJson(t.Node))

	// First look for a Rain section and store constants
	hasRainSection := processRainSection(&t)

	constants := t.Constants

	ctx := &transformContext{
		nodeToTransform: templateNode,
		rootDir:         rootDir,
		t:               t,
		parent:          &node.NodePair{Key: t.Node, Value: t.Node},
		fs:              fs,
	}
	var changed = false
	passes := 0
	maxPasses := 100
	for {
		passes += 1
		// Modules can add new nodes to the template, which
		// breaks s11n.MatchAll, since it expects the length to stay the same.
		// Just start over and transform the whole template again.
		changedThisPass, err := transform(ctx)
		if err != nil {
			return t, err
		}
		if changedThisPass {
			changed = true
		}
		if !changedThisPass {
			break
		}
		if passes > maxPasses {
			return t, errors.New("reached maxPasses while transforming")
		}
	}

	if changed {
		t, err = parse.Node(templateNode)
		if err != nil {
			return t, err
		}
	}

	// Collect Anchors & Replace Alias Nodes
	//
	// 1. find alias nodes and save them in map with anchor name as key
	// 2. replace alias nodes with the actual node
	// 3. Marshal and Unmarshal to resolve new line/column numbers

	v := visitor.NewVisitor(templateNode)
	anchors := make(map[string]*yaml.Node)

	collectAnchors := func(node *visitor.Visitor) {
		yamlNode := node.GetYamlNode()
		if yamlNode.Anchor != "" {
			anchors[yamlNode.Anchor] = yamlNode
			yamlNode.Anchor = ""
		}
	}

	replaceAnchors := func(node *visitor.Visitor) {
		yamlNode := node.GetYamlNode()
		if yamlNode.Kind == yaml.AliasNode {
			if anchor, ok := anchors[yamlNode.Value]; ok {
				*yamlNode = *anchor
			}
		}
	}

	v.Visit(collectAnchors)
	v.Visit(replaceAnchors)

	// Look for ${Rain::ConstantName} in all Sub strings
	if hasRainSection {
		// Note that this rewrites all Subs and might have side effects
		replaceTemplateConstants(templateNode, constants)
	}

	// Marshal and Unmarshal to resolve new line/column numbers

	serialized, err := yaml.Marshal(templateNode)
	if err != nil {
		return t, fmt.Errorf("failed to marshal template: %v", err)
	}

	err = yaml.Unmarshal(serialized, templateNode)
	if err != nil {
		return t, fmt.Errorf("failed to unmarshal template: %v", err)
	}

	// We might lose the Document node here
	retval := cft.Template{}
	if templateNode.Kind == yaml.DocumentNode {
		retval.Node = templateNode
	} else {
		retval.Node = &yaml.Node{Kind: yaml.DocumentNode, Content: make([]*yaml.Node, 0)}
		retval.Node.Content = append(retval.Node.Content, templateNode)
	}

	// Add analytics to Metadata
	if !NoAnalytics {
		metadata, err := retval.GetSection(cft.Metadata)
		if err != nil || metadata == nil {
			metadata = node.AddMap(retval.Node.Content[0], string(cft.Metadata))
		}
		awsToolsMetrics := node.AddMap(metadata, "AWSToolsMetrics")
		a := analytics{
			Version:        config.VERSION,
			HasModules:     HasModules,
			Experimental:   Experimental,
			HasRainSection: HasRainSection,
		}
		s, _ := json.Marshal(&a)
		awsToolsMetrics.Content = append(awsToolsMetrics.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "Rain"})
		awsToolsMetrics.Content = append(awsToolsMetrics.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: string(s)})
	}

	return retval, nil
}

// File opens path as a CloudFormation template and returns a cft.Template
// with assets included as per AWS CLI packaging rules
// and any Rain:: functions used
func File(path string) (cft.Template, error) {
	// config.Debugf("Packaging template: %s\n", path)

	var t cft.Template
	var err error

	if strings.HasSuffix(path, ".pkl") {
		y, err := rainpkl.Yaml(path)
		if err != nil {
			return t, err
		}
		t, err = parse.String(y)
		if err != nil {
			return t, err
		}
	} else {
		t, err = parse.File(path)
		if err != nil {
			config.Debugf("pkg.File unable to parse %v", path)
			return t, err
		}
	}

	return Template(t, filepath.Dir(path), nil)
}
