// Package parse provides functions for parsing
// CloudFormation templates from JSON and YAML inputs.
package parse

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/aws-cloudformation/rain/cfn"
	"github.com/aws-cloudformation/rain/cfn/diff"
	"github.com/aws-cloudformation/rain/cfn/format"

	yaml "github.com/sanathkr/go-yaml"
	yamlwrapper "github.com/sanathkr/yaml"
)

var tags = []string{
	"And",
	"Base64",
	"Cidr",
	"Equals",
	"FindInMap",
	"GetAZs",
	"GetAtt",
	"If",
	"ImportValue",
	"Join",
	"Not",
	"Or",
	"Ref",
	"Select",
	"Split",
	"Sub",
	"Transform",
}

type tagUnmarshalerType struct {
}

var tagUnmarshaler = &tagUnmarshalerType{}

func init() {
	for _, tag := range tags {
		yaml.RegisterTagUnmarshaler("!"+tag, tagUnmarshaler)
	}
}

func (t *tagUnmarshalerType) UnmarshalYAMLTag(tag string, value reflect.Value) reflect.Value {
	prefix := "Fn::"
	if tag == "Ref" || tag == "Condition" {
		prefix = ""
	}
	tag = prefix + tag

	output := reflect.ValueOf(make(map[interface{}]interface{}))
	key := reflect.ValueOf(tag)
	output.SetMapIndex(key, value)

	return output
}

func transform(in map[string]interface{}) map[string]interface{} {
	for k, v := range in {
		if k == "Fn::GetAtt" {
			if s, ok := v.(string); ok {
				value := make([]interface{}, 2)
				for i, part := range strings.SplitN(s, ".", 2) {
					value[i] = part
				}
				in[k] = value
			}
		} else if m, ok := v.(map[string]interface{}); ok {
			in[k] = transform(m)
		}
	}

	return in
}

// Reader returns a cfn.Template parsed from an io.Reader
func Reader(r io.Reader) (cfn.Template, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return cfn.Template{}, fmt.Errorf("Unable to read input: %s", err)
	}

	return String(string(data))
}

// Reader returns a cfn.Template parsed from a file specified by fileName
func File(fileName string) (cfn.Template, error) {
	source, err := ioutil.ReadFile(fileName)
	if err != nil {
		return cfn.Template{}, fmt.Errorf("Unable to read file: %s", err)
	}

	return String(string(source))
}

// Reader returns a cfn.Template parsed from a string
func String(input string) (cfn.Template, error) {
	parsed, err := yamlwrapper.YAMLToJSON([]byte(input))
	if err != nil {
		return cfn.Template{}, fmt.Errorf("Invalid YAML: %s", err)
	}

	var output map[string]interface{}
	err = json.Unmarshal(parsed, &output)
	if err != nil {
		return cfn.Template{}, fmt.Errorf("Invalid YAML: %s", err)
	}

	return Map(output)
}

func Map(input map[string]interface{}) (cfn.Template, error) {
	return cfn.Template(transform(input)), nil
}

// Verify confirms that there is no semantic difference between
// the source cfn.Template and the string representation in output.
// This can be used to ensure that the parse package hasn't done
// anything unexpected to your template.
func Verify(source cfn.Template, output string) error {
	// Check it matches the original
	validate, err := String(output)
	if err != nil {
		return err
	}

	d := source.Diff(validate)

	if d.Mode() != diff.Unchanged {
		return fmt.Errorf("Semantic difference after formatting:\n%s", format.Diff(d, format.Options{Compact: true}))
	}

	return nil
}
