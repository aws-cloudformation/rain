// Package parse provides functions for parsing
// CloudFormation templates from JSON and YAML inputs.
package parse

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/aws-cloudformation/rain/cfn"
	"github.com/aws-cloudformation/rain/cfn/diff"
	"github.com/aws-cloudformation/rain/cfn/format"

	"gopkg.in/yaml.v3"
)

// Reader returns a cfn.Template parsed from an io.Reader
func Reader(r io.Reader) (cfn.Template, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return cfn.Template{}, fmt.Errorf("Unable to read input: %s", err)
	}

	return String(string(data))
}

// File returns a cfn.Template parsed from a file specified by fileName
func File(fileName string) (cfn.Template, error) {
	source, err := ioutil.ReadFile(fileName)
	if err != nil {
		return cfn.Template{}, fmt.Errorf("Unable to read file: %s", err)
	}

	return String(string(source))
}

// String returns a cfn.Template parsed from a string
func String(input string) (cfn.Template, error) {
	var node yaml.Node
	err := yaml.Unmarshal([]byte(input), &node)
	if err != nil {
		return cfn.Template{}, fmt.Errorf("Invalid YAML: %s", err)
	}

	transform(&node)

	var output map[string]interface{}
	err = node.Decode(&output)
	if err != nil {
		return cfn.Template{}, fmt.Errorf("Invalid template: %s", err)
	}

	return cfn.Template(output), nil
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
