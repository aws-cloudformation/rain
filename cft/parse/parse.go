// Package parse provides functions for parsing
// CloudFormation templates from JSON and YAML inputs.
package parse

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/diff"

	"gopkg.in/yaml.v3"
)

// Reader returns a cft.Template parsed from an io.Reader
func Reader(r io.Reader) (*cft.Template, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return &cft.Template{}, fmt.Errorf("unable to read input: %s", err)
	}

	return String(string(data))
}

// File returns a cft.Template parsed from a file specified by fileName
func File(fileName string) (*cft.Template, error) {
	if fileName == "-" {
		return Reader(os.Stdin)
	}

	source, err := os.ReadFile(fileName)
	if err != nil {
		return &cft.Template{}, fmt.Errorf("unable to read file: %s", err)
	}

	t, err := String(string(source))
	if err != nil {
		return t, err
	}
	t.FileName = fileName
	tokens := strings.Split(fileName, "/")
	if len(tokens) > 1 {
		t.Name = tokens[len(tokens)-1]
	} else {
		t.Name = fileName
	}
	return t, nil
}

// Map returns a cft.Template parsed from a map[string]interface{}
func Map(input map[string]interface{}) (*cft.Template, error) {
	var node yaml.Node
	err := node.Encode(input)
	if err != nil {
		return &cft.Template{}, err
	}

	return Node(&node)
}

// String returns a cft.Template parsed from a string
func String(input string) (*cft.Template, error) {
	var n yaml.Node
	err := yaml.Unmarshal([]byte(input), &n)
	if err != nil {
		return &cft.Template{}, fmt.Errorf("invalid YAML: %s", err)
	}

	return Node(&n)
}

// Node returns a cft.Template parse from a *yaml.Node
func Node(n *yaml.Node) (*cft.Template, error) {
	err := NormalizeNode(n)
	return &cft.Template{Node: n}, err
}

// Verify confirms that there is no semantic difference between
// the source cft.Template and the string representation in output.
// This can be used to ensure that the parse package hasn't done
// anything unexpected to your template.
func Verify(source *cft.Template, output string) error {
	// Check it matches the original
	validate, err := String(output)
	if err != nil {
		return err
	}

	d := diff.New(source, validate)
	if d.Mode() != diff.Unchanged {
		return fmt.Errorf("semantic difference after formatting:\n%s", d.Format(false))
	}

	return nil
}
