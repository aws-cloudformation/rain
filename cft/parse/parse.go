// Package parse provides functions for parsing
// CloudFormation templates from JSON and YAML inputs.
package parse

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/diff"

	"gopkg.in/yaml.v3"
)

// Reader returns a cft.Template parsed from an io.Reader
func Reader(r io.Reader) (cft.Template, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return cft.Template{}, fmt.Errorf("Unable to read input: %s", err)
	}

	return String(string(data))
}

// File returns a cft.Template parsed from a file specified by fileName
func File(fileName string) (cft.Template, error) {
	source, err := ioutil.ReadFile(fileName)
	if err != nil {
		return cft.Template{}, fmt.Errorf("Unable to read file: %s", err)
	}

	return String(string(source))
}

// Map returns a cft.Template parsed from a map[string]interface{}
func Map(input map[string]interface{}) (cft.Template, error) {
	var node yaml.Node
	err := node.Encode(input)
	if err != nil {
		return cft.Template{}, err
	}

	return Node(&node)
}

// String returns a cft.Template parsed from a string
func String(input string) (cft.Template, error) {
	var node yaml.Node
	err := yaml.Unmarshal([]byte(input), &node)
	if err != nil {
		return cft.Template{}, fmt.Errorf("Invalid YAML: %s", err)
	}

	return Node(&node)
}

// Node returns a cft.Template parse from a *yaml.Node
func Node(node *yaml.Node) (cft.Template, error) {
	TransformNode(node)
	return cft.Template{Node: node}, nil
}

// Verify confirms that there is no semantic difference between
// the source cft.Template and the string representation in output.
// This can be used to ensure that the parse package hasn't done
// anything unexpected to your template.
func Verify(source cft.Template, output string) error {
	// Check it matches the original
	validate, err := String(output)
	if err != nil {
		return err
	}

	d := diff.New(source, validate)
	if d.Mode() != diff.Unchanged {
		return fmt.Errorf("Semantic difference after formatting:\n%s", d.Format(false))
	}

	return nil
}
