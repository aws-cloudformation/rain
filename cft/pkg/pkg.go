// Package pkg provides functionality similar to the AWS CLI cloudformation package command
// but has greater flexibility, allowing content to be included anywhere in a template
//
// To include content into your templates, use any of the following either as YAML tags
// or as one-property objects, much as AWS instrinsic functions are used, e.g. "Fn::Join"
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
	"path/filepath"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// Must be set to true to enable !Rain::Module
var Experimental bool

type transformContext struct {
	nodeToTransform *yaml.Node
	rootDir         string // Using normal files
	t               cft.Template
	parent          *node.NodePair
	fs              *embed.FS // Used by build with embedded filesytem
}

func transform(ctx *transformContext) (bool, error) {
	changed := false

	// registry is a map of functions defined in rain.go
	for path, fn := range registry {
		//config.Debugf("transform path: %v, nodeToTransform:\n%v", path,
		//	node.ToSJson(nodeToTransform))
		for found := range s11n.MatchAll(ctx.nodeToTransform, path) {
			nodeParent := node.GetParent(found, ctx.nodeToTransform, nil)
			nodeParent.Parent = ctx.parent
			c, err := fn(&directiveContext{found, ctx.rootDir, ctx.t, nodeParent, ctx.fs})
			if err != nil {
				config.Debugf("Error packaging template: %s\n", err)
				return false, err
			}

			changed = changed || c
		}
	}

	return changed, nil
}

// Template returns t with assets included as per AWS CLI packaging rules
// and any Rain:: functions used.
// rootDir must be passed in so that any included assets can be loaded from the same directory
func Template(t cft.Template, rootDir string, fs *embed.FS) (cft.Template, error) {
	templateNode := t.Node

	// j, _ := json.MarshalIndent(t.Node, "", "  ")
	// config.Debugf("Original template: %v", string(j))

	ctx := &transformContext{
		nodeToTransform: templateNode,
		rootDir:         rootDir,
		t:               t,
		parent:          nil,
		fs:              fs,
	}
	changed, err := transform(ctx)
	if err != nil {
		return t, err
	}

	// j, _ = json.MarshalIndent(templateNode, "", "  ")
	// config.Debugf("Transformed template: %v", string(j))

	if changed {
		t, err = parse.Node(templateNode)
		if err != nil {
			return t, err
		}
	}

	// Encode and Decode to resolve anchors
	var decoded interface{}

	config.Debugf("About to decode:\n%v", node.ToSJson(templateNode))
	err = templateNode.Decode(&decoded)
	if err != nil {
		return t, err
	}

	err = templateNode.Encode(&decoded)
	if err != nil {
		return t, err
	}

	// We lose the Document node here
	retval := cft.Template{}
	retval.Node = &yaml.Node{Kind: yaml.DocumentNode, Content: make([]*yaml.Node, 0)}
	retval.Node.Content = append(retval.Node.Content, templateNode)

	return retval, err
}

// File opens path as a CloudFormation template and returns a cft.Template
// with assets included as per AWS CLI packaging rules
// and any Rain:: functions used
func File(path string) (cft.Template, error) {
	// config.Debugf("Packaging template: %s\n", path)

	t, err := parse.File(path)
	if err != nil {
		config.Debugf("pkg.File unable to parse %v", path)
		return t, err
	}

	return Template(t, filepath.Dir(path), nil)
}
