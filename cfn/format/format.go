// Package format provides functions for formatting the types found in the cfn package.
package format

import (
	"sort"

	"github.com/aws-cloudformation/rain/cfn"
	"github.com/aws-cloudformation/rain/cfn/diff"
	"github.com/aws-cloudformation/rain/cfn/value"
)

// Style represents a style offered by a given formatter
type Style int

const (
	YAML Style = iota
	JSON
)

// Options represents a collection of formatting options
type Options struct {
	Style   Style
	Compact bool
}

// Value returns a string representation of any given value
// as either JSON or YAML depending on options.Style
func Value(data value.Value, options Options) string {
	return newEncoder(options, data).format()
}

// Anything returns a string representation of any given value
// as either JSON or YAML depending on options.Style
func Anything(data interface{}, options Options) string {
	return newEncoder(options, value.New(data)).format()
}

// Template returns a string representation of a cfn.Template
// as either JSON or YAML depending on options.Style
func Template(t cfn.Template, options Options) string {
	return newEncoder(options, value.New(t)).format()
}

// Diff returns a string representation of a diff.Diff.
// options.Style is currently ignored and the format will be annotated YAML
func Diff(d diff.Diff, options Options) string {
	if options.Compact && d.Mode() == diff.Unchanged {
		return ""
	}

	return formatDiff(d, []interface{}{}, !options.Compact)
}

// SortKeys sorts the given keys
// based on their location within a CloudFormation template
// as given by the path parameter
func SortKeys(keys []string, path []interface{}) []string {
	data := make(map[string]interface{}, len(keys))
	for _, key := range keys {
		data[key] = nil
	}

	p := encoder{
		Options: Options{},
		value:   value.New(data),
		path:    path,
	}
	p.get()

	sorted := p.sortKeys()

	// Because some of the formatters rely on template data that's missing
	// go through the original keys and append any that have been removed
	missing := make([]string, 0)
	for _, orig := range keys {
		found := false

		for _, newKey := range sorted {
			if newKey == orig {
				found = true
				break
			}
		}

		if !found {
			missing = append(missing, orig)
		}
	}

	sort.Strings(missing)

	return append(sorted, missing...)
}
