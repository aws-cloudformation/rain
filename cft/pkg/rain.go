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
	_, err = transform(&contentNode, filepath.Dir(path), t, nil)
	if err != nil {
		return false, err
	}

	// Unwrap from the document node
	*n = *contentNode.Content[0]
	return true, nil
}

func includeEnv(n *yaml.Node, root string, t cft.Template, parent node.NodePair) (bool, error) {
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
	_, pathOption := s11n.GetMapValue(n.Content[1], "Path")
	if pathOption != nil && pathOption.Kind == yaml.MappingNode {
		config.Debugf("includeS3Object Path is a map: %v", node.ToJson(pathOption))
		if pathOption.Content[0].Value == "Ref" {
			// If this S3 directive is embedded in a module, we need to look at the
			// resource in the parent template and get the Property with the same name
			config.Debugf("Path Ref %v, root is %v, parent.Key: %v, parent.Value: %v",
				pathOption.Content[1].Value, root, node.ToJson(parent.Key), node.ToJson(parent.Value))
			// How do we get a reference to the parent template resource?
			config.Debugf("t is %v", node.ToJson(t.Node))
			// t is the parent template that references the module, but we don't know
			// what resource within the template to reference
			if parent.Parent != nil {
				config.Debugf("parent.Parent is not nil: Key: %v, Value: %v",
					node.ToJson(parent.Parent.Key), node.ToJson(parent.Parent.Value))
				moduleParentMap := parent.Parent.Value
				_, moduleParentProps := s11n.GetMapValue(moduleParentMap, "Properties")
				if moduleParentProps != nil {
					_, pathProp := s11n.GetMapValue(moduleParentProps, pathOption.Content[1].Value)
					// Replace the Ref with the value
					node.SetMapValue(n.Content[1], "Path", node.Clone(pathProp))
					config.Debugf("After replacing path node, options: %v", node.ToJson(n.Content[1]))
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

	config.Debugf("includeS3Object options: %v", options)

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
