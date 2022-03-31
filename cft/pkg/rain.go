package pkg

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/cft/parse"
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

type rainFunc func(*yaml.Node, string) (bool, error)

var registry = make(map[string]rainFunc)

func init() {
	registry["**/*|Rain::Embed"] = includeString
	registry["**/*|Rain::Include"] = includeLiteral
	registry["**/*|Rain::S3Http"] = includeS3Http
	registry["**/*|Rain::S3"] = includeS3
}

func includeString(n *yaml.Node, root string) (bool, error) {
	content, _, err := expectFile(n, root)
	if err != nil {
		return false, err
	}

	n.Encode(strings.TrimSpace(string(content)))

	return true, nil
}

func includeLiteral(n *yaml.Node, root string) (bool, error) {
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
	_, err = transform(&node, filepath.Dir(path))
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

func includeS3Object(n *yaml.Node, root string) (bool, error) {
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

func includeS3Http(n *yaml.Node, root string) (bool, error) {
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

func includeS3URI(n *yaml.Node, root string) (bool, error) {
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

func includeS3(n *yaml.Node, root string) (bool, error) {
	// Figure out if we're a string or an object
	if len(n.Content) != 2 {
		return false, errors.New("expected exactly one key")
	}

	if n.Content[1].Kind == yaml.ScalarNode {
		return includeS3URI(n, root)
	} else if n.Content[1].Kind == yaml.MappingNode {
		return includeS3Object(n, root)
	}

	return true, nil
}
