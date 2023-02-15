// This file contains implementations for `!Rain::` directives
package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
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

type rainFunc func(*yaml.Node, string, cft.Template) (bool, error)

var registry = make(map[string]rainFunc)

func init() {
	registry["**/*|Rain::Embed"] = includeString
	registry["**/*|Rain::Include"] = includeLiteral
	registry["**/*|Rain::S3Http"] = includeS3Http
	registry["**/*|Rain::S3"] = includeS3
	registry["**/*|Rain::Module"] = module
}

func includeString(n *yaml.Node, root string, t cft.Template) (bool, error) {
	content, _, err := expectFile(n, root)
	if err != nil {
		return false, err
	}

	n.Encode(strings.TrimSpace(string(content)))

	return true, nil
}

func includeLiteral(n *yaml.Node, root string, t cft.Template) (bool, error) {
	content, path, err := expectFile(n, root)
	if err != nil {
		return false, err
	}

	var node yaml.Node
	err = yaml.Unmarshal(content, &node)
	if err != nil {
		return false, err
	}

	// Transform
	parse.TransformNode(&node)
	_, err = transform(&node, filepath.Dir(path), t)
	if err != nil {
		return false, err
	}

	// Unwrap from the document node
	*n = *node.Content[0]
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

func includeS3Object(n *yaml.Node, root string, t cft.Template) (bool, error) {
	if n.Kind != yaml.MappingNode || len(n.Content) != 2 {
		return false, errors.New("expected a map")
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

func includeS3Http(n *yaml.Node, root string, t cft.Template) (bool, error) {
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

func includeS3URI(n *yaml.Node, root string, t cft.Template) (bool, error) {
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

func includeS3(n *yaml.Node, root string, t cft.Template) (bool, error) {
	// Figure out if we're a string or an object
	if len(n.Content) != 2 {
		return false, errors.New("expected exactly one key")
	}

	if n.Content[1].Kind == yaml.ScalarNode {
		return includeS3URI(n, root, t)
	} else if n.Content[1].Kind == yaml.MappingNode {
		return includeS3Object(n, root, t)
	}

	return true, nil
}

// Convert the module into a node for the packaged template
func processModule(module *yaml.Node, outputNode *yaml.Node, t cft.Template) (bool, error) {
	config.Debugf("processModule module: %v", module)

	outputNode.Content = make([]*yaml.Node, 0)

	if module.Kind != yaml.DocumentNode {
		return false, errors.New("expected module to be a DocumentNode")
	}

	curNode := module.Content[0] // ScalarNode !!map

	_, resources := s11n.GetMapValue(curNode, "Resources")

	if resources == nil {
		return false, errors.New("expected the module to have a Resources section")
	}

	_, moduleExtension := s11n.GetMapValue(resources, "ModuleExtension")
	if moduleExtension == nil {
		return false, errors.New("expected the module to have a single ModuleExtension resource")
	}

	// Get additional resources and add them to the output
	for i, resource := range resources.Content {
		tag := resource.ShortTag()
		config.Debugf("resource: %v, %v", tag, resource.Value)
		if resource.Kind == yaml.MappingNode {
			name := resources.Content[i-1].Value
			if name != "ModuleExtension" {
				// This is an additional resource to be added
				config.Debugf("Adding additional resource to output node: %v", name)
				outputNode.Content = append(outputNode.Content, resources.Content[i-1])
				outputNode.Content = append(outputNode.Content, resource)
			}
		}
	}

	return true, nil
}

// Type: !Rain::Module
func module(n *yaml.Node, root string, t cft.Template) (bool, error) {

	if len(n.Content) != 2 {
		return false, errors.New("expected !Rain::Module <URI>")
	}

	// TODO - remove these after testing
	config.Debugf("module node: %v", n)
	config.Debugf("module node.Content: %v", n.Content)
	for i, c := range n.Content {
		config.Debugf("module n.Content[%v]: %v %v", i, c.Kind, c.Value)
	}
	config.Debugf("module root: %v", root)

	uri := n.Content[1].Value
	// Is this a local file or a URL?
	if strings.HasPrefix(uri, "file://") {
		// Read the local file
		content, path, err := expectFile(n, root)
		if err != nil {
			return false, err
		}

		// Parse the file
		var node yaml.Node
		err = yaml.Unmarshal(content, &node)
		if err != nil {
			return false, err
		}

		// Transform
		parse.TransformNode(&node)
		_, err = transform(&node, filepath.Dir(path), t)
		if err != nil {
			return false, err
		}

		// Create a new node to represent the processed module
		var outputNode yaml.Node
		_, err = processModule(&node, &outputNode, t)
		if err != nil {
			return false, err
		}

		// TODO - remove these after testing
		j, _ := json.Marshal(outputNode)
		config.Debugf("module outputNode: %v", string(j))

		j, _ = json.Marshal(t.Node)
		config.Debugf("t: %v", string(j))

		// Find the resource node in the template
		_, resourceNode := s11n.GetMapValue(t.Node.Content[0], "Resources")
		if resourceNode == nil {
			return false, errors.New("expected template to have Resources")
		}

		// Insert into the template
		for i, c := range outputNode.Content {
			resourceNode.Content = append(resourceNode.Content, outputNode.Content[i])
		}
	} else if strings.HasPrefix(uri, "https://") {
		// Download the file and then parse it
		// TODO
	} else {
		return false, errors.New("expected either file://path or https://path")
	}

	return true, nil

}
