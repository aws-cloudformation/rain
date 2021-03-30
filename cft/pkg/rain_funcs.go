package pkg

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
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
		"Resources/*|Type==AWS::Serverless::Function/Properties/CodeUri":                                     wrapS3ZipUri,
		"Resources/*|Type==AWS::Serverless::LayerVersion/Properties/ContentUri":                              wrapS3ZipUri,
		"Resources/*|Type==AWS::CloudFormation::Stack/Properties/TemplateURL":                                wrapTemplate,
		"Resources/*|Type==AWS::Serverless::Application/Properties/Location":                                 wrapTemplate,
		"Resources/*|Type==AWS::Lambda::Function/Properties/Code":                                            wrapObject("S3Bucket", "S3Key", true),
		"Resources/*|Type==AWS::ApiGateway::RestApi/Properties/BodyS3Location":                               wrapObject("Bucket", "Key", false),
		"Resources/*|Type==AWS::ElasticBeanstalk::ApplicationVersion/Properties/SourceBundle":                wrapObject("S3Bucket", "S3Key", false),
		"Resources/*|Type==AWS::Lambda::LayerVersion/Properties/Content":                                     wrapObject("S3Bucket", "S3Key", true),
		"Resources/*|Type==AWS::StepFunctions::StateMachine/Properties/DefinitionS3Location":                 wrapObject("Bucket", "Key", false),
	}
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

func wrapS3Uri(n *yaml.Node) bool {
	if n.Kind != yaml.ScalarNode {
		return false
	}

	newNode, changed := handleS3(s3Options{
		Path:   n.Value,
		Format: s3Uri,
	})

	if changed {
		*n = *newNode
	}

	return changed
}

func wrapObject(bucket, key string, forceZip bool) rainFunc {
	return func(n *yaml.Node) bool {
		if n.Kind != yaml.ScalarNode {
			return false
		}

		newNode, changed := handleS3(s3Options{
			Path:           n.Value,
			Format:         s3Object,
			BucketProperty: bucket,
			KeyProperty:    key,
			Zip:            forceZip,
		})

		if changed {
			*n = *newNode
		}

		return changed
	}
}

func wrapTemplate(n *yaml.Node) bool {
	if n.Kind != yaml.ScalarNode {
		return false
	}

	content, err := readFile(n.Value)
	if err != nil {
		return false
	}

	tmpl, err := parse.String(string(content))
	if err != nil {
		return false
	}

	tmpl, err = Template(tmpl)
	if err != nil {
		return false
	}

	f, err := ioutil.TempFile(os.TempDir(), "*.template")
	if err != nil {
		return false
	}

	defer os.Remove(f.Name())

	if _, err := f.Write([]byte(format.String(tmpl, format.Options{}))); err != nil {
		return false
	}

	if err := f.Close(); err != nil {
		return false
	}

	newNode, changed := handleS3(s3Options{
		Path:   f.Name(),
		Format: s3Http,
	})

	if changed {
		*n = *newNode
	}

	return changed
}
