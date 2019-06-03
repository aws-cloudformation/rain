package parse

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"

	"github.com/google/go-cmp/cmp"
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

func Read(r io.Reader) (map[string]interface{}, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("Unable to read input: %s", err)
	}

	return ReadString(string(data))
}

func ReadFile(fileName string) (map[string]interface{}, error) {
	source, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("Unable to read file: %s", err)
	}

	return ReadString(string(source))
}

func ReadString(input string) (map[string]interface{}, error) {
	parsed, err := yamlwrapper.YAMLToJSON([]byte(input))
	if err != nil {
		return nil, fmt.Errorf("Invalid YAML: %s", err)
	}

	var output map[string]interface{}
	err = json.Unmarshal(parsed, &output)
	if err != nil {
		return nil, fmt.Errorf("Invalid YAML: %s", err)
	}

	return output, nil
}

func VerifyOutput(source map[string]interface{}, output string) error {
	// Check it matches the original
	validate, err := ReadString(output)
	if err != nil {
		return err
	}

	if diff := cmp.Diff(source, validate); diff != "" {
		return fmt.Errorf("Semantic difference after formatting:\n%s", diff)
	}

	return nil
}
