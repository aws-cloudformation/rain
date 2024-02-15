// This file contains implementations for `!Rain::` directives
package pkg

import (
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
}

// A !Rain directive implementation
type rainFunc func(n *yaml.Node, rootDir string, t cft.Template, parent node.NodePair) (bool, error)

var registry = make(map[string]rainFunc)

func init() {
	registry["**/*|Rain::Embed"] = includeString
	registry["**/*|Rain::Include"] = includeLiteral
	registry["**/*|Rain::Env"] = includeEnv
	registry["**/*|Rain::S3Http"] = includeS3Http
	registry["**/*|Rain::S3"] = includeS3
	registry["**/*|Rain::Module"] = module
}

func includeString(n *yaml.Node,
	root string, t cft.Template, parent node.NodePair) (bool, error) {
	content, _, err := expectFile(n, root)
	if err != nil {
		return false, err
	}

	n.Encode(strings.TrimSpace(string(content)))

	return true, nil
}

func includeLiteral(n *yaml.Node, root string, t cft.Template, parent node.NodePair) (bool, error) {
	content, path, err := expectFile(n, root)
	if err != nil {
		return false, err
	}

	var contentNode yaml.Node
	err = yaml.Unmarshal(content, &contentNode)
	if err != nil {
		return false, err
	}

	// Transform
	parse.TransformNode(&contentNode)
	ctx := &transformContext{
		nodeToTransform: &contentNode,
		rootDir:         filepath.Dir(path),
		t:               t,
		parent:          nil,
		fs:              nil,
	}
	_, err = transform(ctx)
	if err != nil {
		return false, err
	}

	// Unwrap from the document node
	*n = *contentNode.Content[0]
	return true, nil
}

func includeEnv(n *yaml.Node, root string, t cft.Template, parent node.NodePair) (bool, error) {
	config.Debugf("includeEnv n: %v", node.ToSJson(n))
	name, err := expectString(n)
	if err != nil {
		return false, err
	}
	val, present := os.LookupEnv(name)
	if !present {
		return false, fmt.Errorf("missing environmental variable %q", name)
	}
	var newNode yaml.Node
	newNode.Encode(val)
	if err != nil {
		return false, err
	}
	*n = newNode
	return true, nil
}

func handleS3(root string, options s3Options) (*yaml.Node, error) {
	s, err := upload(root, options.Path, options.Zip)
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

		n.Encode(out)
	case s3URI:
		n.Encode(s.URI())
	case s3Http:
		n.Encode(s.HTTP())
	default:
		return nil, fmt.Errorf("unexpected S3 output format: %s", options.Format)
	}

	return &n, nil
}

func includeS3Object(n *yaml.Node, root string, t cft.Template, parent node.NodePair) (bool, error) {
	if n.Kind != yaml.MappingNode || len(n.Content) != 2 {
		return false, errors.New("expected a map")
	}

	// Check to see if the Path is a Ref.
	// The only valid use case is if the !Rain::S3 directive is inside a module,
	// and the Ref points to one of the properties set in the parent template
	_, pathOption, _ := s11n.GetMapValue(n.Content[1], "Path")
	if pathOption != nil && pathOption.Kind == yaml.MappingNode {
		if pathOption.Content[0].Value == "Ref" {
			if parent.Parent != nil {
				moduleParentMap := parent.Parent.Value
				_, moduleParentProps, _ := s11n.GetMapValue(moduleParentMap, "Properties")
				if moduleParentProps != nil {
					_, pathProp, _ := s11n.GetMapValue(moduleParentProps, pathOption.Content[1].Value)
					if pathProp != nil {
						// Replace the Ref with the value
						node.SetMapValue(n.Content[1], "Path", node.Clone(pathProp))
					} else {
						config.Debugf("expected Properties to have Path")
					}
				} else {
					config.Debugf("expected parent resource to have Properties")
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

	newNode, err := handleS3(root, options)
	if err != nil {
		return false, err
	}

	*n = *newNode

	return true, nil
}

func includeS3Http(n *yaml.Node, root string, t cft.Template, parent node.NodePair) (bool, error) {
	path, err := expectString(n)
	if err != nil {
		return false, err
	}

	newNode, err := handleS3(root, s3Options{
		Path:   path,
		Format: s3Http,
	})

	if err != nil {
		return false, err
	}

	*n = *newNode

	return true, nil
}

func includeS3URI(n *yaml.Node, root string, t cft.Template, parent node.NodePair) (bool, error) {
	path, err := expectString(n)
	if err != nil {
		return false, err
	}

	newNode, err := handleS3(root, s3Options{
		Path:   path,
		Format: s3URI,
	})

	if err != nil {
		return false, err
	}

	*n = *newNode

	return true, nil
}

func includeS3(n *yaml.Node, root string, t cft.Template, parent node.NodePair) (bool, error) {
	// Figure out if we're a string or an object
	if len(n.Content) != 2 {
		return false, errors.New("expected exactly one key")
	}

	if n.Content[1].Kind == yaml.ScalarNode {
		return includeS3URI(n, root, t, parent)
	} else if n.Content[1].Kind == yaml.MappingNode {
		return includeS3Object(n, root, t, parent)
	}

	return true, nil
}
