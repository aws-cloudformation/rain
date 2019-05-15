package diff

// TODO: Calculate whether an added map/slice is all new (>>>) or has changes (|||)

import (
	"fmt"
	"sort"
	"strings"

	"github.com/awslabs/aws-cloudformation-template-formatter/format"
)

// We'll use these later
var colorMap = map[string]string{
	"=== ": "\033[33m",
	"||| ": "\033[33m",
	">>> ": "\033[32m",
	"<<< ": "\033[31m",
}

var modeStrings = map[mode]string{
	added:   ">>> ",
	removed: "<<< ",
	changed: "||| ",
}

const indent = "  "

func Format(d diff) string {
	switch v := d.(type) {
	case diffSlice:
		return formatSlice(v)
	case diffMap:
		return formatMap(v)
	case diffValue:
		f := format.NewFormatter()
		f.SetCompact()
		return f.Format(v.value)
	}

	panic("Unimplemented comparison")
}

func formatSlice(d diffSlice) string {
	output := strings.Builder{}

	for i, v := range d {
		m := v.mode()

		if m != unchanged {
			// Always treat a value as added
			if _, isValue := v.(diffValue); isValue {
				output.WriteString(modeStrings[added])
			} else {
				output.WriteString(modeStrings[m])
			}

			output.WriteString(fmt.Sprintf("[%d]", i))

			if m == removed {
				output.WriteString("\n")
			} else {
				output.WriteString(":")
				output.WriteString(formatSub(v))
			}
		}
	}

	return output.String()
}

func formatMap(d diffMap) string {
	output := strings.Builder{}

	// Sort the keys
	keys := make([]string, 0)

	for k, _ := range d {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		v := d[k]
		m := v.mode()

		if m != unchanged {
			// Always treat a value as added
			if _, isValue := v.(diffValue); isValue {
				output.WriteString(modeStrings[added])
			} else {
				output.WriteString(modeStrings[m])
			}

			output.WriteString(k)

			if m == removed {
				output.WriteString("\n")
			} else {
				output.WriteString(":")
				output.WriteString(formatSub(v))
			}
		}
	}

	return output.String()
}

func formatSub(d diff) string {
	// Format the element
	formatted := strings.Split(Format(d), "\n")

	// It's a scalar
	if len(formatted) == 1 {
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

	_, isValue := d.(diffValue)

	if len(parts) == 0 {
		panic("Something's gone wrong")
	} else {
		output.WriteString("\n")
		for _, part := range parts {
			if isValue {
				part = modeStrings[added] + indent + part
			} else {
				part = part[:4] + indent + part[4:]
			}

			output.WriteString(part)
			output.WriteString("\n")
		}
	}

	return output.String()
}
