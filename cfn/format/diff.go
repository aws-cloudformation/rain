package format

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/aws-cloudformation/rain/cfn/diff"
	"github.com/aws-cloudformation/rain/cfn/value"
)

const indent = "  "

func formatDiff(d diff.Diff, path []interface{}, long bool) string {
	switch v := d.(type) {
	case diff.Slice:
		return formatSlice(v, path, long)
	case diff.Map:
		return formatMap(v, path, long)
	case diff.Value:
		return Value(value.New(v.Value()), Options{Compact: true})
	}

	panic(fmt.Sprintf("Unexpected %#v\n", d))
}

func stubValue(v diff.Value) string {
	switch v.Value().(type) {
	case map[string]interface{}:
		return "{...}"
	case []interface{}:
		return "[...]"
	default:
		return "..."
	}
}

func formatSlice(s diff.Slice, path []interface{}, long bool) string {
	output := strings.Builder{}

	for i, v := range s {
		m := v.Mode()

		if !long && m == diff.Unchanged {
			continue
		}

		output.WriteString(fmt.Sprintf("%s [%d]:", m, i))

		if !long && (m == diff.Removed || m == diff.Unchanged) {
			output.WriteString(" " + stubValue(v.(diff.Value)) + "\n")
		} else {
			output.WriteString(formatSub(v, append(path, i), long))
		}
	}

	return output.String()
}

func formatMap(m diff.Map, path []interface{}, long bool) string {
	output := strings.Builder{}

	keys := m.Keys()
	keys = SortKeys(keys, path)

	for _, k := range keys {
		v := m[k]
		m := v.Mode()

		if !long && m == diff.Unchanged {
			continue
		}

		output.WriteString(fmt.Sprintf("%s %s:", m, k))

		if !long && (m == diff.Removed || m == diff.Unchanged) {
			output.WriteString(" " + stubValue(v.(diff.Value)) + "\n")
		} else {
			output.WriteString(formatSub(v, append(path, k), long))
		}
	}

	return output.String()
}

func formatSub(d diff.Diff, path []interface{}, long bool) string {
	// Format the element
	formatted := formatDiff(d, path, long)

	v, isValue := d.(diff.Value)
	if isValue {
		k := reflect.ValueOf(v.Value()).Kind()

		if k != reflect.Array && k != reflect.Map && k != reflect.Slice {
			return fmt.Sprintf(" %s\n", formatted)
		}
	}

	// Trim out blank lines
	parts := make([]string, 0)
	for _, part := range strings.Split(formatted, "\n") {
		if strings.TrimSpace(part) != "" {
			parts = append(parts, part)
		}
	}

	output := strings.Builder{}

	output.WriteString("\n")
	for _, part := range parts {
		if isValue {
			part = fmt.Sprintf("%s %s%s", v.Mode(), indent, part)
		} else {
			part = part[:len(diff.Added.String())] + indent + part[len(diff.Added.String()):]
		}

		output.WriteString(part)
		output.WriteString("\n")
	}

	return output.String()
}
