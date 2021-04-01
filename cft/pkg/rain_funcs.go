package pkg

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
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

type rainFunc func(*yaml.Node, string) (bool, error)

var registry map[string]rainFunc

func init() {
	registry = map[string]rainFunc{
		"**/*|Rain::Embed":   includeString,
		"**/*|Rain::Include": includeLiteral,
		"**/*|Rain::S3Http":  includeS3Http,
		"**/*|Rain::S3":      includeS3,
		"Resources/*|Type==AWS::Serverless::Api/Properties/DefinitionUri":                                    wrapS3Uri,
		"Resources/*|Type==AWS::AppSync::GraphQLSchema/Properties/DefinitionS3Location":                      wrapS3Uri,
		"Resources/*|Type==AWS::AppSync::Resolver/Properties/RequestMappingTemplateS3Location":               wrapS3Uri,
		"Resources/*|Type==AWS::AppSync::Resolver/Properties/ResponseMappingTemplateS3Location":              wrapS3Uri,
		"Resources/*|Type==AWS::AppSync::FunctionConfiguration/Properties/RequestMappingTemplateS3Location":  wrapS3Uri,
		"Resources/*|Type==AWS::AppSync::FunctionConfiguration/Properties/ResponseMappingTemplateS3Location": wrapS3Uri,
		"Resources/*|Type==AWS::ServerlessRepo::Application/Properties/ReadmeUrl":                            wrapS3Uri,
		"Resources/*|Type==AWS::ServerlessRepo::Application/Properties/LicenseUrl":                           wrapS3Uri,
		"Resources/*|Type==AWS::Glue::Job/Properties/Command/ScriptLocation":                                 wrapS3Uri,
		"Resources/*|Type==AWS::Serverless::Function/Properties/CodeUri":                                     wrapS3ZipURI,
		"Resources/*|Type==AWS::Serverless::LayerVersion/Properties/ContentUri":                              wrapS3ZipURI,
		"Resources/*|Type==AWS::CloudFormation::Stack/Properties/TemplateURL":                                wrapTemplate,
		"Resources/*|Type==AWS::Serverless::Application/Properties/Location":                                 wrapTemplate,
		"Resources/*|Type==AWS::Lambda::Function/Properties/Code":                                            wrapObject("S3Bucket", "S3Key", true),
		"Resources/*|Type==AWS::ApiGateway::RestApi/Properties/BodyS3Location":                               wrapObject("Bucket", "Key", false),
		"Resources/*|Type==AWS::ElasticBeanstalk::ApplicationVersion/Properties/SourceBundle":                wrapObject("S3Bucket", "S3Key", false),
		"Resources/*|Type==AWS::Lambda::LayerVersion/Properties/Content":                                     wrapObject("S3Bucket", "S3Key", true),
		"Resources/*|Type==AWS::StepFunctions::StateMachine/Properties/DefinitionS3Location":                 wrapObject("Bucket", "Key", false),
	}
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

func handleS3(options s3Options) (*yaml.Node, error) {
	s, err := upload(options.Path, options.Zip)
	if err != nil {
		return nil, err
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
			return nil, errors.New("Missing BucketProperty or KeyProperty")
		}

		out := map[string]string{
			options.BucketProperty: s.bucket,
			options.KeyProperty:    s.key,
		}

		n.Encode(out)
	case s3Uri:
		n.Encode(s.URI())
	case s3Http:
		n.Encode(s.HTTP())
	default:
		return nil, fmt.Errorf("Unexpected S3 output format: %s", options.Format)
	}

	return &n, nil
}

func includeS3Object(n *yaml.Node, root string) (bool, error) {
	if n.Kind != yaml.MappingNode || len(n.Content) != 2 {
		return false, errors.New("Expected a map")
	}

	// Parse the options
	var options s3Options
	err := n.Content[1].Decode(&options)
	if err != nil {
		return false, err
	}

	newNode, err := handleS3(options)
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

	newNode, err := handleS3(s3Options{
		Path:   path,
		Format: s3Http,
	})

	if err != nil {
		return false, err
	}

	*n = *newNode

	return true, nil
}

func includeS3Uri(n *yaml.Node, root string) (bool, error) {
	path, err := expectString(n)
	if err != nil {
		return false, err
	}

	newNode, err := handleS3(s3Options{
		Path:   path,
		Format: s3Uri,
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
		return false, errors.New("Expected exactly one key")
	}

	if n.Content[1].Kind == yaml.ScalarNode {
		return includeS3Uri(n, root)
	} else if n.Content[1].Kind == yaml.MappingNode {
		return includeS3Object(n, root)
	}

	return true, nil
}

func wrapS3(n *yaml.Node, options s3Options) bool {
	newNode, err := handleS3(options)
	if err != nil {
		return false
	}

	*n = *newNode

	return true
}

func wrapS3ZipURI(n *yaml.Node, root string) (bool, error) {
	if n.Kind != yaml.ScalarNode {
		// No need to error - this could be valid
		return false, nil
	}

	return wrapS3(n, s3Options{
		Path:   n.Value,
		Format: s3Uri,
		Zip:    true,
	}), nil
}

func wrapS3Uri(n *yaml.Node, root string) (bool, error) {
	if n.Kind != yaml.ScalarNode {
		// No need to error - this could be valid
		return false, nil
	}

	return wrapS3(n, s3Options{
		Path:   n.Value,
		Format: s3Uri,
	}), nil
}

func wrapObject(bucket, key string, forceZip bool) rainFunc {
	return func(n *yaml.Node, root string) (bool, error) {
		if n.Kind != yaml.ScalarNode {
			// No need to error - this could be valid
			return false, nil
		}

		return wrapS3(n, s3Options{
			Path:           n.Value,
			Format:         s3Object,
			BucketProperty: bucket,
			KeyProperty:    key,
			Zip:            forceZip,
		}), nil
	}
}

func wrapTemplate(n *yaml.Node, root string) (bool, error) {
	if n.Kind != yaml.ScalarNode {
		// No need to error - this could be valid
		return false, nil
	}

	if strings.HasPrefix(n.Value, "http://") || strings.HasPrefix(n.Value, "https://") {
		return false, nil // Already http
	}

	path := n.Value
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}

	tmpl, err := File(path)
	if err != nil {
		return false, err
	}

	f, err := ioutil.TempFile(os.TempDir(), "*.template")
	if err != nil {
		return false, err
	}
	defer os.Remove(f.Name())

	if _, err := f.Write([]byte(format.String(tmpl, format.Options{}))); err != nil {
		return false, err
	}

	if err := f.Close(); err != nil {
		return false, err
	}

	return wrapS3(n, s3Options{
		Path:   f.Name(),
		Format: s3Http,
	}), nil
}
