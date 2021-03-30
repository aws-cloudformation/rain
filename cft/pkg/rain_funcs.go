package pkg

import (
	"strings"

	"github.com/aws-cloudformation/rain/internal/config"
	"gopkg.in/yaml.v3"
)

type rainFunc func(*yaml.Node) bool

var registry = map[string]rainFunc{
	"**/*|Rain::Embed":   includeString,
	"**/*|Rain::Include": includeLiteral,
	"**/*|Rain::S3Http":  includeS3Http,
	"**/*|Rain::S3":      includeS3,
	"Resources/*|Type==AWS::Serverless::Function/Properties/CodeUri": serverlessFunction,
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

func includeS3Http(n *yaml.Node) bool {
	path, ok := expectString(n)
	if !ok {
		return false
	}

	s, err := upload(path)
	if err != nil {
		config.Debugf("Error uploading '%s': %s", path, err)
		return false
	}

	n.Encode(s.Http())

	return true
}

func includeS3Uri(n *yaml.Node) bool {
	path, ok := expectString(n)
	if !ok {
		return false
	}

	s, err := upload(path)
	if err != nil {
		config.Debugf("Error uploading '%s': %s", path, err)
		return false
	}

	n.Encode(s.Uri())

	return true
}

func includeS3Object(n *yaml.Node) bool {
	props, ok := expectProps(n, "Path", "BucketProperty", "KeyProperty")
	if !ok {
		return false
	}

	path := props["Path"]

	s, err := upload(path)
	if err != nil {
		config.Debugf("Error uploading '%s': %s", path, err)
		return false
	}

	out := map[string]string{
		props["BucketProperty"]: s.bucket,
		props["KeyProperty"]:    s.key,
	}

	n.Encode(out)

	return true
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

func serverlessFunction(n *yaml.Node) bool {
	if n.Kind != yaml.ScalarNode {
		return false
	}

	var newNode yaml.Node
	err := newNode.Encode(map[string]interface{}{
		"Rain::S3": n.Value,
	})
	if err != nil {
		config.Debugf("Error converting to Rain::S3: %s", err)
		return false
	}

	changed := includeS3Uri(&newNode)
	if changed {
		*n = newNode
	}

	return changed
}
