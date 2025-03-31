// Package pkg provides functionality similar to the AWS CLI cloudformation
// package command but has greater flexibility, allowing content to be included
// anywhere in a template
//
// To include content into your templates, use any of the following either as
// YAML tags or as one-property objects, much as AWS intrinsic functions are
// used, e.g. "Fn::Join"
//
// `Rain::Include`: insert the content of the file into the template directly.
// The file must be in YAML or JSON format.
//
// `Rain::Env`: inserts environmental variable value into the template as a
// string. Variable must be set.
//
// `Rain::Embed`: insert the content of the file as a string
//
// `Rain::S3Http`: uploads the file or directory (zipping it first) to S3 and
// returns the HTTP URI (i.e.  `https://bucket.s3.region.amazonaws.com/key`)
//
// `Rain::S3`: a string value uploads the file or directory (zipping it
// first) to S3 and returns the S3 URI (i.e. `s3://bucket/key`)
//
// `Rain::S3`: an object with the following properties
//
//	`Path`: path to the file or directory. If a directory is supplied, it will
//	be zipped before uploading to S3 `BucketProperty`: Name of returned
//	property that will contain the bucket name `KeyProperty`: Name of returned
//	property that will contain the object key `VersionProperty`: (optional)
//	Name of returned property that will contain the object version
//
// `Rain::Module`: Supply a URL to a rain module
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

const AWSToolsMetrics = "AWSToolsMetrics"
const Rain = "Rain"

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
	t               *cft.Template
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
func processRainSection(t *cft.Template, rootDir string, fs *embed.FS) error {
	t.Constants = make(map[string]*yaml.Node)
	rainNode, err := t.GetSection(cft.Rain)
	if err != nil {
		// This is okay, not all templates have a Rain section
		return nil
	}

	HasRainSection = true

	err = processConstants(t, rainNode)
	if err != nil {
		return err
	}

	err = processPackages(t, t.Node)
	if err != nil {
		return err
	}

	// Now remove the Rain node from the template
	t.RemoveSection(cft.Rain)

	return nil
}

// Template returns t with assets included as per AWS CLI packaging rules
// and any Rain:: functions used.
// rootDir must be passed in so that any included assets can be loaded from the same directory
func Template(t *cft.Template, rootDir string, fs *embed.FS) (*cft.Template, error) {
	var err error

	//config.Debugf("Original template short: %v", node.ToSJson(t.Node))
	//config.Debugf("Original template long: %v", node.ToJson(t.Node))

	// First look for a Rain section and store constants
	err = processRainSection(t, rootDir, fs)
	if err != nil {
		return nil, fmt.Errorf("failed to process Rain section: %v", err)
	}

	// See if those sections are defined at the top level
	err = processAddedSections(t, t.Node.Content[0], rootDir, fs, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to process added sections: %v", err)
	}

	ctx := &transformContext{
		nodeToTransform: t.Node,
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
			return nil, err
		}
		if changedThisPass {
			changed = true
		}
		if !changedThisPass {
			break
		}
		if passes > maxPasses {
			return nil, errors.New("reached maxPasses while transforming")
		}
	}

	if changed {
		t, err = parse.Node(t.Node)
		if err != nil {
			return nil, err
		}
	}

	// Collect Anchors & Replace Alias Nodes
	//
	// 1. find alias nodes and save them in map with anchor name as key
	// 2. replace alias nodes with the actual node
	// 3. Marshal and Unmarshal to resolve new line/column numbers

	v := visitor.NewVisitor(t.Node)
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

	//// Look for ${Rain::ConstantName} and ${Const::ConstantName}
	//if t.HasSection(cft.Rain) || t.HasSection(cft.Constants) {
	//	// Note that this rewrites all Subs and might have side effects
	//	err = replaceTemplateConstants(templateNode, constants)
	//	if err != nil {
	//		return nil, err
	//	}
	//}

	// Marshal and Unmarshal to resolve new line/column numbers

	serialized, err := yaml.Marshal(t.Node)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal template: %v", err)
	}

	err = yaml.Unmarshal(serialized, t.Node)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %v", err)
	}

	// We might lose the Document node here
	retval := &cft.Template{}
	if t.Node.Kind == yaml.DocumentNode {
		retval.Node = t.Node
	} else {
		retval.Node = &yaml.Node{Kind: yaml.DocumentNode, Content: make([]*yaml.Node, 0)}
		retval.Node.Content = append(retval.Node.Content, t.Node)
	}

	// Add analytics to Metadata
	if !NoAnalytics {
		metadata, err := retval.GetSection(cft.Metadata)
		if err != nil || metadata == nil {
			metadata = node.AddMap(retval.Node.Content[0], string(cft.Metadata))
		}
		_, awsToolsMetrics, _ := s11n.GetMapValue(metadata, AWSToolsMetrics)
		if awsToolsMetrics == nil {
			awsToolsMetrics = node.AddMap(metadata, AWSToolsMetrics)
		}
		a := analytics{
			Version:        config.VERSION,
			HasModules:     HasModules,
			Experimental:   Experimental,
			HasRainSection: HasRainSection,
		}
		s, _ := json.Marshal(&a)
		_, rain, _ := s11n.GetMapValue(awsToolsMetrics, Rain)
		rainNode := &yaml.Node{Kind: yaml.ScalarNode, Value: string(s)}
		if rain == nil {
			awsToolsMetrics.Content = append(awsToolsMetrics.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: Rain})
			awsToolsMetrics.Content = append(awsToolsMetrics.Content, rainNode)
		} else {
			*rain = *rainNode
		}
	}

	return retval, nil
}

// File opens path as a CloudFormation template and returns a cft.Template
// with assets included as per AWS CLI packaging rules
// and any Rain:: functions used
func File(path string) (*cft.Template, error) {
	// config.Debugf("Packaging template: %s\n", path)

	var t *cft.Template
	var err error

	if strings.HasSuffix(path, ".pkl") {
		y, err := rainpkl.Yaml(path)
		if err != nil {
			return nil, err
		}
		t, err = parse.String(y)
		if err != nil {
			return nil, err
		}
	} else {
		t, err = parse.File(path)
		if err != nil {
			config.Debugf("pkg.File unable to parse %v", path)
			return nil, err
		}
	}

	return Template(t, filepath.Dir(path), nil)
}
