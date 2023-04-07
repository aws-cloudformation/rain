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
	node := t.Node

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
	lastPartWasComment := false
	lastLineWasEmpty := false
	startMultilineIndent := -1

	for _, part := range parts {

		trimmedPart := strings.TrimLeft(part, " ")
		indent = len(part) - len(trimmedPart)

		// Leave lines alone if they are in a multiline block
		// Note: CloudFormation does not comply with the YAML spec. It treats > just like |
		// https://yaml-multiline.info/
		// https://stackoverflow.com/questions/3790454/how-do-i-break-a-string-in-yaml-over-multiple-lines
		// https://yaml.org/spec/1.2-old/spec.html#id2760844
		isMultiline := false
		if startMultilineIndent > -1 {
			if startMultilineIndent <= indent {
				startMultilineIndent = -1
			} else {
				isMultiline = true
			}
		}
		trimmedRight := strings.TrimRight(part, " ")
		if strings.HasSuffix(trimmedRight, "|") || strings.HasSuffix(trimmedRight, ">") {
			startMultilineIndent = indent
		}

		isComment := false
		if len(part) > 0 && strings.HasPrefix(trimmedPart, "#") {
			isComment = true
		}

		isEmpty := len(part) == 0 // This should never be true

		// Add lines between 1st and 2nd level properties, except for comments
		if indent <= lastIndent && (indent == 0 || indent == 2) {
			if !lastPartWasComment && !isMultiline {
				// If the last line was a comment, don't newline here,
				// since we want the comment to stick to the thing it was above
				result.WriteString("\n")
				lastLineWasEmpty = true
			}
		}

		// Put a line break above first/only comment lines
		if !lastPartWasComment && isComment && !lastLineWasEmpty && !isMultiline {
			result.WriteString("\n")
		}

		result.WriteString(part)
		result.WriteString("\n")

		lastIndent = indent
		lastPartWasComment = isComment
		lastLineWasEmpty = isEmpty
	}

	out := strings.TrimSpace(result.String())

	if opt.JSON {
		out = convertToJSON(out)
	}

	return out + "\n"
}
