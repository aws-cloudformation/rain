package format

import (
	"fmt"
	"strings"
)

type encoder struct {
	formatter      Formatter
	data           value
	path           []interface{}
	currentValue   interface{}
	currentComment string
}

func newEncoder(formatter Formatter, data value) encoder {
	p := encoder{
		formatter: formatter,
		data:      data,
		path:      make([]interface{}, 0),
	}

	p.get()

	return p
}

func (p *encoder) get() {
	p.currentValue = p.data.Get(p.path)
	p.currentComment = p.data.GetComment(p.path)
}

func (p *encoder) push(key interface{}) {
	p.path = append(p.path, key)
	p.get()
}

func (p *encoder) pop() {
	p.path = p.path[:len(p.path)-1]
	p.get()
}

func (p encoder) indent(in string) string {
	indenter := "  "

	if p.formatter.style == JSON {
		indenter = "    "
	}
	parts := strings.Split(in, "\n")

	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = indenter + part
		}
	}

	if p.formatter.style == JSON {
		return strings.Join(parts, "\n")
	}

	return strings.TrimLeft(strings.Join(parts, "\n"), " ")
}

func (p encoder) formatIntrinsic(key string) string {
	p.push(key)
	defer p.pop()

	if p.formatter.style == JSON {
		return p.format()
	}

	shortKey := strings.Replace(key, "Fn::", "", 1)

	fmtValue := p.format()

	switch p.currentValue.(type) {
	case map[string]interface{}, []interface{}:
		return fmt.Sprintf("!%s\n  %s", shortKey, p.indent(fmtValue))
	default:
		return fmt.Sprintf("!%s %s", shortKey, fmtValue)
	}
}

func (p encoder) formatMap(data map[string]interface{}) string {
	if len(data) == 0 {
		return "{}"
	}

	keys := sortKeys(data, p.path)

	parts := make([]string, len(keys))

	for i, key := range keys {
		value := data[key]

		p.push(key)
		fmtValue := p.format()

		if p.formatter.style == JSON {
			fmtValue = fmt.Sprintf("%q: %s", key, fmtValue)
			if i < len(keys)-1 {
				fmtValue += ","
			}

			isScalar := true
			switch value.(type) {
			case map[string]interface{}, []interface{}:
				isScalar = false
			}

			if p.currentComment != "" && isScalar {
				fmtValue += "  // " + p.currentComment
			}
		} else {
			needsIndent := false

			// CloudFormation requires string keys
			key = formatString(key)

			switch v := value.(type) {
			case map[string]interface{}:
				if iKey, ok := intrinsicKey(v); ok {
					fmtValue = p.formatIntrinsic(iKey)
				} else {
					needsIndent = true
				}
			case []interface{}:
				if fmtValue != "[]" {
					needsIndent = true
				}
			}

			if needsIndent {
				if p.currentComment != "" {
					fmtValue = fmt.Sprintf("%s:  # %s\n  %s", key, p.currentComment, p.indent(fmtValue))
				} else {
					fmtValue = fmt.Sprintf("%s:\n  %s", key, p.indent(fmtValue))
				}
			} else {
				if p.currentComment != "" {
					fmtValue = fmt.Sprintf("%s: %s  # %s", key, fmtValue, p.currentComment)
				} else {
					fmtValue = fmt.Sprintf("%s: %s", key, fmtValue)
				}
			}
		}

		parts[i] = fmtValue

		p.pop()
	}

	// Double gap for top-level elements
	joiner := "\n"
	if !p.formatter.compact && len(p.path) <= 1 {
		joiner = "\n\n"
	}

	if p.formatter.style == JSON {
		if p.currentComment != "" {
			return "{  // " + p.currentComment + "\n" + p.indent(strings.Join(parts, joiner)) + "\n}"
		}

		return "{\n" + p.indent(strings.Join(parts, joiner)) + "\n}"
	}

	output := strings.Join(parts, joiner)

	// Add a top-level comment for yaml
	if p.currentComment != "" && len(p.path) == 0 {
		output = "# " + p.currentComment + "\n" + output
	}

	return output
}

func (p encoder) formatList(data []interface{}) string {
	if len(data) == 0 {
		return "[]"
	}

	parts := make([]string, len(data))

	for i := range data {
		p.push(i)
		fmtValue := p.format()

		if p.formatter.style == JSON {
			parts[i] = p.indent(fmtValue)
		} else {
			parts[i] = fmt.Sprintf("- %s", p.indent(fmtValue))
		}

		if p.currentComment != "" {
			if p.formatter.style == JSON {
				parts[i] += "  // " + p.currentComment
			} else {
				parts[i] += "  # " + p.currentComment
			}
		}

		p.pop()
	}

	if p.formatter.style == JSON {
		if p.currentComment != "" {
			return "[  // " + p.currentComment + "\n" + strings.Join(parts, ",\n") + "\n]"
		}

		return "[\n" + strings.Join(parts, ",\n") + "\n]"
	}

	return strings.Join(parts, "\n")
}

func (p encoder) format() string {
	switch v := p.currentValue.(type) {
	case map[string]interface{}:
		return p.formatMap(v)
	case []interface{}:
		return p.formatList(v)
	case string:
		if p.formatter.style == JSON {
			return fmt.Sprintf("%q", v)
		}

		return formatString(v)
	default:
		return fmt.Sprint(v)
	}
}
