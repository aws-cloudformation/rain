package diff

// TODO: Calculate whether an added map/slice is all new (>>>) or has changes (|||)

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/aws-cloudformation/rain/format"
)

var yaml format.Formatter

func init() {
	yaml = format.New(format.Options{
		Style:   format.YAML,
		Compact: true,
	})
}

const indent = "  "

func Format(d Diff, long bool) string {
	return formatDiff(d, make([]interface{}, 0), long)
}

func formatDiff(d Diff, path []interface{}, long bool) string {
	switch v := d.(type) {
	case diffSlice:
		return formatSlice(v, path, long)
	case diffMap:
		return formatMap(v, path, long)
	case diffValue:
		return yaml.Format(v.value)
	}

	panic(fmt.Sprintf("Unexpected %#v\n", d))
}

func stubValue(v diffValue) string {
	switch v.value.(type) {
	case map[string]interface{}:
		return "{...}"
	case []interface{}:
		return "[...]"
	default:
		return "..."
	}
}

func formatSlice(d diffSlice, path []interface{}, long bool) string {
	output := strings.Builder{}

	for i, v := range d {
		m := v.Mode()

		if !long && m == Unchanged {
			continue
		}

		output.WriteString(fmt.Sprintf("%s[%d]:", m.String(), i))

		if m == Removed || m == Unchanged {
			output.WriteString(" " + stubValue(v.(diffValue)) + "\n")
		} else {
			output.WriteString(formatSub(v, append(path, i), long))
		}
	}

	return output.String()
}

func formatMap(d diffMap, path []interface{}, long bool) string {
	output := strings.Builder{}

	keys := d.Keys()
	keys = format.SortKeys(keys, path)

	for _, k := range keys {
		v := d[k]
		m := v.Mode()

		if !long && m == Unchanged {
			continue
		}

		output.WriteString(fmt.Sprintf("%s%s:", m.String(), k))

		if m == Removed || m == Unchanged {
			output.WriteString(" " + stubValue(v.(diffValue)) + "\n")
		} else {
			output.WriteString(formatSub(v, append(path, k), long))
		}
	}

	return output.String()
}

func formatSub(d Diff, path []interface{}, long bool) string {
	// Format the element
	formatted := strings.Split(formatDiff(d, path, long), "\n")

	v, isValue := d.(diffValue)
	if isValue {
		k := reflect.ValueOf(v.value).Kind()

		if k != reflect.Array && k != reflect.Map && k != reflect.Slice {
			// It's a scalar
			return fmt.Sprintf(" %s\n", formatted[0])
		}
	} else if len(formatted) == 1 {
		// It's a scalar
		return fmt.Sprintf(" %s\n", formatted[0])
	}

	// Trim out blank lines
	parts := make([]string, 0)
	for _, part := range formatted {
		if strings.TrimSpace(part) != "" {
			parts = append(parts, part)
		}
	}

	output := strings.Builder{}

	output.WriteString("\n")
	for _, part := range parts {
		if isValue {
			part = Added.String() + indent + part
		} else {
			part = part[:len(Added.String())] + indent + part[len(Added.String()):]
		}

		output.WriteString(part)
		output.WriteString("\n")
	}

	return output.String()
}
