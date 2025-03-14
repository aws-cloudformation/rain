package pkg

// This file contains implementations for `!Rain::` directives

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

type s3Format string

const (
	s3URI    s3Format = "URI"
	s3Http   s3Format = "Http"
	s3Object s3Format = "Object"
)

type s3Options struct {
	Path           string   `yaml:"Path"`
	BucketProperty string   `yaml:"BucketProperty"`
	KeyProperty    string   `yaml:"KeyProperty"`
	Zip            bool     `yaml:"Zip"`
	Format         s3Format `yaml:"Format"`
	Run            string   `yaml:"Run"`
	Extension      string   `yaml:"Extension"`
}

type directiveContext struct {
	n       *yaml.Node
	rootDir string
	t       *cft.Template
	parent  node.NodePair
	fs      *embed.FS

	// baseUri is the base path for downloading submodules
	baseUri string
}

// A !Rain directive implementation
type directiveFunc func(ctx *directiveContext) (bool, error)

var registry = make(map[string]directiveFunc)

func init() {
	registry["**/*|Rain::Embed"] = includeString
	registry["**/*|Rain::Include"] = includeLiteral
	registry["**/*|Rain::Env"] = includeEnv
	registry["**/*|Rain::S3Http"] = includeS3Http
	registry["**/*|Rain::S3"] = includeS3
	registry["**/*|Rain::Module"] = module
	registry["**/*|Rain::Constant"] = rainConstant

	// Don't forget to also add new items to cft/tags.go
}

func includeString(ctx *directiveContext) (bool, error) {

	content, _, err := expectFile(ctx.n, ctx.rootDir)
	if err != nil {
		return false, err
	}

	err = ctx.n.Encode(strings.TrimSpace(string(content)))
	if err != nil {
		return false, err
	}

	return true, nil
}

func includeLiteral(ctx *directiveContext) (bool, error) {
	content, path, err := expectFile(ctx.n, ctx.rootDir)
	if err != nil {
		return false, err
	}

	var contentNode yaml.Node
	err = yaml.Unmarshal(content, &contentNode)
	if err != nil {
		return false, err
	}

	// Transform
	err = parse.NormalizeNode(&contentNode)
	if err != nil {
		return false, err
	}

	_, err = transform(&transformContext{
		nodeToTransform: &contentNode,
		rootDir:         filepath.Dir(path),
		t:               ctx.t,
		parent:          nil,
		fs:              nil,
	})
	if err != nil {
		return false, err
	}

	// Unwrap from the document node
	*ctx.n = *contentNode.Content[0]
	return true, nil
}

func includeEnv(ctx *directiveContext) (bool, error) {
	name, err := expectString(ctx.n)
	if err != nil {
		return false, err
	}
	val, present := os.LookupEnv(name)
	if !present {
		return false, fmt.Errorf("missing environmental variable %q", name)
	}
	var newNode yaml.Node
	err = newNode.Encode(val)
	if err != nil {
		return false, err
	}
	if err != nil {
		return false, err
	}
	*ctx.n = newNode
	return true, nil
}

func handleS3(root string, options s3Options) (*yaml.Node, error) {

	// Check to see if we need to run a build command first
	if options.Run != "" {
		relativePath := filepath.Join(".", root, options.Run)
		absPath, absErr := filepath.Abs(relativePath)
		if absErr != nil {
			config.Debugf("filepath.Abs failed? %s", absErr)
			return nil, absErr
		}
		cmd := exec.Command(absPath)
		var stdout strings.Builder
		var stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		cmd.Dir = root
		err := cmd.Run()
		if err != nil {
			config.Debugf("s3Option Run %s failed with %s: %s",
				options.Run, err, stderr.String())
			return nil, err
		}
	}

	s, err := upload(root, options.Path, options.Zip, options.Extension)
	if err != nil {
		return nil, err
	}

	if options.Format == "" {
		if options.BucketProperty != "" && options.KeyProperty != "" {
			options.Format = s3Object
		} else {
			options.Format = s3URI
		}
	}

	var n yaml.Node

	switch options.Format {
	case s3Object:
		if options.BucketProperty == "" || options.KeyProperty == "" {
			return nil, errors.New("missing BucketProperty or KeyProperty")
		}

		out := map[string]string{
			options.BucketProperty: s.bucket,
			options.KeyProperty:    s.key,
		}

		err := n.Encode(out)
		if err != nil {
			return nil, err
		}
	case s3URI:
		err := n.Encode(s.URI())
		if err != nil {
			return nil, err
		}
	case s3Http:
		err := n.Encode(s.HTTP())
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unexpected S3 output format: %s", options.Format)
	}

	return &n, nil
}

func includeS3Object(ctx *directiveContext) (bool, error) {

	n := ctx.n
	parent := ctx.parent
	if n.Kind != yaml.MappingNode || len(n.Content) != 2 {
		return false, errors.New("expected a map")
	}

	// Check to see if any of the properties is a Ref.
	// The only valid use case is if the !Rain::S3 directive is inside a module,
	// and the Ref points to one of the properties set in the parent template
	for i := 0; i < len(n.Content[1].Content); i += 2 {
		s3opt := n.Content[1].Content[i+1]
		name := n.Content[1].Content[i].Value
		if s3opt.Kind == yaml.MappingNode {
			if s3opt.Content[0].Value == "Ref" {
				if parent.Parent != nil {
					moduleParentMap := parent.Parent.Value
					_, moduleParentProps, _ := s11n.GetMapValue(moduleParentMap, "Properties")
					if moduleParentProps != nil {
						_, parentProp, _ := s11n.GetMapValue(moduleParentProps, s3opt.Content[1].Value)
						if parentProp != nil {
							// Replace the Ref with the value
							node.SetMapValue(n.Content[1], name, node.Clone(parentProp))
						} else {
							config.Debugf("expected Properties to have Path")
						}
					} else {
						config.Debugf("expected parent resource to have Properties")
						config.Debugf("moduleParentMap: %s", node.ToSJson(moduleParentMap))
					}
				}
			}
		}
	}

	// Parse the options
	var options s3Options
	err := n.Content[1].Decode(&options)
	if err != nil {
		return false, err
	}

	newNode, err := handleS3(ctx.rootDir, options)
	if err != nil {
		return false, err
	}

	*n = *newNode

	return true, nil
}

func includeS3Http(ctx *directiveContext) (bool, error) {
	path, err := expectString(ctx.n)
	if err != nil {
		return false, err
	}

	newNode, err := handleS3(ctx.rootDir, s3Options{
		Path:   path,
		Format: s3Http,
	})

	if err != nil {
		return false, err
	}

	*ctx.n = *newNode

	return true, nil
}

func includeS3URI(ctx *directiveContext) (bool, error) {
	path, err := expectString(ctx.n)
	if err != nil {
		return false, err
	}

	newNode, err := handleS3(ctx.rootDir, s3Options{
		Path:   path,
		Format: s3URI,
	})

	if err != nil {
		return false, err
	}

	*ctx.n = *newNode

	return true, nil
}

func includeS3(ctx *directiveContext) (bool, error) {
	n := ctx.n
	root := ctx.rootDir
	t := ctx.t
	parent := ctx.parent
	// Figure out if we're a string or an object
	if len(n.Content) != 2 {
		return false, errors.New("expected exactly one key")
	}

	if n.Content[1].Kind == yaml.ScalarNode {
		return includeS3URI(&directiveContext{n, root, t, parent, nil, ""})
	} else if n.Content[1].Kind == yaml.MappingNode {
		return includeS3Object(&directiveContext{n, root, t, parent, nil, ""})
	}

	return true, nil
}
