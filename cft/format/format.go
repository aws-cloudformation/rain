// Package format contains functionality to render a cft.Template
// into YAML or JSON
package format

import (
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"gopkg.in/yaml.v3"
)

// Options contains options for formatting cfn templates
type Options struct {
	// JSON determines whether the outputs will be JSON (true) or YAML (false)
	JSON bool

	// Unsorted will cause the formatter to leave the ordering of template elements
	// as in the original template if true.
	// If false, the formatter will rearrange the template elements into
	// canonical order.
	Unsorted bool
}

// String returns a string representation of the supplied cft.Template
func String(t cft.Template, opt Options) string {
	node := &t.Node

	buf := strings.Builder{}
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	node = formatNode(node)

	if !opt.Unsorted {
		node = orderTemplate(node)
	}

	err := enc.Encode(node)
	if err != nil {
		panic(err)
	}

	parts := strings.Split(strings.TrimSpace(buf.String()), "\n")
	result := strings.Builder{}

	lastIndent := 0
	indent := 0

	for _, part := range parts {
		indent = len(part) - len(strings.TrimLeft(part, " "))

		// Add spaces between 1st and 2nd level properties
		if indent <= lastIndent && (indent == 0 || indent == 2) {
			result.WriteString("\n")
		}

		result.WriteString(part)
		result.WriteString("\n")

		lastIndent = indent
	}

	out := result.String()

	if opt.JSON {
		out = convertToJSON(out)
	}

	return strings.TrimSpace(out)
}
