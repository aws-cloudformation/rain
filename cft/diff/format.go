package diff

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const indent = "  "

// Format returns a pretty-printed representation of the slice
func (s slice) Format(long bool) string {
	return formatSlice(s, []interface{}{}, long)
}

// Format returns a pretty-printed representation of the dmap
func (m dmap) Format(long bool) string {
	return formatMap(m, []interface{}{}, long)
}

// Format returns a pretty-printed representation of the value
func (v value) Format(long bool) string {
	buf := strings.Builder{}

	e := yaml.NewEncoder(&buf)
	e.SetIndent(2)

	err := e.Encode(v.value())
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(buf.String())
}

func stubValue(v value) string {
	switch v.value().(type) {
	case map[string]interface{}:
		return "{...}"
	case []interface{}:
		return "[...]"
	default:
		return "..."
	}
}

func formatDiff(d Diff, path []interface{}, long bool) string {
	switch v := d.(type) {
	case slice:
		return formatSlice(v, path, long)
	case dmap:
		return formatMap(v, path, long)
	case value:
		return v.Format(long)
	default:
		panic(fmt.Errorf("Unexpected type '%T'", d))
	}
}

func formatSlice(s slice, path []interface{}, long bool) string {
	output := strings.Builder{}

	for i, v := range s {
		m := v.Mode()

		if !long && m == Unchanged {
			continue
		}

		output.WriteString(fmt.Sprintf("%s [%d]:", m, i))

		if !long && (m == Removed || m == Unchanged) {
			output.WriteString(" " + stubValue(v.(value)) + "\n")
		} else {
			output.WriteString(formatSub(v, append(path, i), long))
		}
	}

	return output.String()
}

func formatMap(m dmap, path []interface{}, long bool) string {
	output := strings.Builder{}

	keys := m.keys()
	sort.Strings(keys)

	for _, k := range keys {
		v := m[k]
		m := v.Mode()

		if !long && m == Unchanged {
			continue
		}

		output.WriteString(fmt.Sprintf("%s %s:", m, k))

		if !long && (m == Removed || m == Unchanged) {
			output.WriteString(" " + stubValue(v.(value)) + "\n")
		} else {
			output.WriteString(formatSub(v, append(path, k), long))
		}
	}

	return output.String()
}

func formatSub(d Diff, path []interface{}, long bool) string {
	// Format the element
	formatted := formatDiff(d, path, long)

	v, isValue := d.(value)
	if isValue {
		k := reflect.ValueOf(v.value()).Kind()

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
			part = part[:len(Added.String())] + indent + part[len(Added.String()):]
		}

		output.WriteString(part)
		output.WriteString("\n")
	}

	return output.String()
}
