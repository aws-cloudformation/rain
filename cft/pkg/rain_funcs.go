package pkg

import (
	"strings"

	"github.com/aws-cloudformation/rain/internal/config"
	"gopkg.in/yaml.v3"
)

type s3Format string

const (
	s3Uri    s3Format = "Uri"
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

type rainFunc func(*yaml.Node) bool

var registry = map[string]rainFunc{
	"**/*|Rain::Embed":   includeString,
	"**/*|Rain::Include": includeLiteral,
	"**/*|Rain::S3Http":  includeS3Http,
	"**/*|Rain::S3":      includeS3,
	"Resources/*|Type==AWS::Serverless::Function/Properties/CodeUri":  wrapS3ZipUri,
	"Resources/*|Type==AWS::Serverless::Api/Properties/DefinitionUri": wrapS3ZipUri,
}

func includeString(n *yaml.Node) bool {
	path, ok := expectString(n)
	if !ok {
		return false
	}

	content, err := readFile(path)
	if err != nil {
		config.Debugf("Error reading file '%s': %s", path, err)
		return false
	}

	n.Encode(strings.TrimSpace(string(content)))

	return true
}

func includeLiteral(n *yaml.Node) bool {
	path, ok := expectString(n)
	if !ok {
		return false
	}

	content, err := readFile(path)
	if err != nil {
		config.Debugf("Error reading file '%s': %s", path, err)
		return false
	}

	err = yaml.Unmarshal(content, n)
	if err != nil {
		config.Debugf("Error unmarshaling file '%s': %s", path, err)
		return false
	}

	// Unwrap from the document node
	*n = *n.Content[0]

	return true
}

func handleS3(options s3Options) (*yaml.Node, bool) {
	s, err := upload(options.Path, options.Zip)
	if err != nil {
		config.Debugf("Error uploading '%s': %s", options.Path, err)
		return nil, false
	}

	if options.Format == "" {
		if options.BucketProperty != "" && options.KeyProperty != "" {
			options.Format = s3Object
		} else {
			options.Format = s3Uri
		}
	}

	var n yaml.Node

	switch options.Format {
	case s3Object:
		if options.BucketProperty == "" || options.KeyProperty == "" {
			return nil, false
		}

		out := map[string]string{
			options.BucketProperty: s.bucket,
			options.KeyProperty:    s.key,
		}

		n.Encode(out)
	case s3Uri:
		n.Encode(s.Uri())
	case s3Http:
		n.Encode(s.Http())
	default:
		return nil, false
	}

	return &n, true
}

func includeS3Object(n *yaml.Node) bool {
	if n.Kind != yaml.MappingNode || len(n.Content) != 2 {
		return false
	}

	// Parse the options
	var options s3Options
	err := n.Content[1].Decode(&options)
	if err != nil {
		return false
	}

	newNode, changed := handleS3(options)
	if changed {
		*n = *newNode
	}

	return changed
}

func includeS3Http(n *yaml.Node) bool {
	path, ok := expectString(n)
	if !ok {
		return false
	}

	newNode, changed := handleS3(s3Options{
		Path:   path,
		Format: s3Http,
	})

	if changed {
		*n = *newNode
	}

	return changed
}

func includeS3Uri(n *yaml.Node) bool {
	path, ok := expectString(n)
	if !ok {
		return false
	}

	newNode, changed := handleS3(s3Options{
		Path:   path,
		Format: s3Uri,
	})

	if changed {
		*n = *newNode
	}

	return changed
}

func includeS3(n *yaml.Node) bool {
	// Figure out if we're a string or an object

	if len(n.Content) != 2 {
		return false
	}

	if n.Content[1].Kind == yaml.ScalarNode {
		return includeS3Uri(n)
	} else if n.Content[1].Kind == yaml.MappingNode {
		return includeS3Object(n)
	}

	return false
}

func wrapS3ZipUri(n *yaml.Node) bool {
	if n.Kind != yaml.ScalarNode {
		return false
	}

	newNode, changed := handleS3(s3Options{
		Path:   n.Value,
		Format: s3Uri,
		Zip:    true,
	})

	if changed {
		*n = *newNode
	}

	return changed
}
