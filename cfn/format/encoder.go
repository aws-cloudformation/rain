package format

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cfn"
	"github.com/aws-cloudformation/rain/cfn/value"
)

type encoder struct {
	Options
	value        value.Interface
	path         []interface{}
	currentValue value.Interface
}

func newEncoder(options Options, data value.Interface) encoder {
	p := encoder{
		Options: options,
		value:   data,
		path:    make([]interface{}, 0),
	}

	p.get()

	return p
}

func (p *encoder) get() {
	p.currentValue = p.value.Get(p.path...)
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

	if p.Style == JSON {
		indenter = "    "
	}
	parts := strings.Split(in, "\n")

	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = indenter + part
		}
	}

	if p.Style == JSON {
		return strings.Join(parts, "\n")
	}

	return strings.TrimLeft(strings.Join(parts, "\n"), " ")
}

func (p encoder) formatIntrinsic(key string) string {
	p.push(key)
	defer p.pop()

	if p.Style == JSON {
		return p.format()
	}

	shortKey := strings.Replace(key, "Fn::", "", 1)

	fmtValue := p.format()

	switch v := p.currentValue.Value().(type) {
	case []interface{}:
		// Deal with GetAtt
		if shortKey == "GetAtt" {
			parts := make([]string, len(v))
			for i, part := range v {
				parts[i] = fmt.Sprint(part)
			}

			return fmt.Sprintf("!%s %s", shortKey, strings.Join(parts, "."))
		}

		return fmt.Sprintf("!%s\n  %s", shortKey, p.indent(fmtValue))
	case map[string]interface{}:
		return fmt.Sprintf("!%s\n  %s", shortKey, p.indent(fmtValue))
	default:
		return fmt.Sprintf("!%s %s", shortKey, fmtValue)
	}
}

func (p encoder) formatMap(data map[string]interface{}) string {
	if len(data) == 0 {
		return "{}"
	}

	keys := p.sortKeys()

	parts := make([]string, len(keys))

	for i, key := range keys {
		value := data[key]

		p.push(key)
		fmtValue := p.format()

		if p.Style == JSON {
			fmtValue = fmt.Sprintf("%q: %s", key, fmtValue)
			if i < len(keys)-1 {
				fmtValue += ","
			}

			isScalar := true
			switch value.(type) {
			case map[string]interface{}, []interface{}:
				isScalar = false
			}

			if p.currentValue.Comment() != "" && isScalar {
				fmtValue += "  // " + p.currentValue.Comment()
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
				if p.currentValue.Comment() != "" {
					fmtValue = fmt.Sprintf("%s:  # %s\n  %s", key, p.currentValue.Comment(), p.indent(fmtValue))
				} else {
					fmtValue = fmt.Sprintf("%s:\n  %s", key, p.indent(fmtValue))
				}
			} else {
				if p.currentValue.Comment() != "" {
					fmtValue = fmt.Sprintf("%s: %s  # %s", key, fmtValue, p.currentValue.Comment())
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
	if !p.Compact && len(p.path) <= 1 {
		joiner = "\n\n"
	}

	if p.Style == JSON {
		if p.currentValue.Comment() != "" {
			return "{  // " + p.currentValue.Comment() + "\n" + p.indent(strings.Join(parts, joiner)) + "\n}"
		}

		return "{\n" + p.indent(strings.Join(parts, joiner)) + "\n}"
	}

	output := strings.Join(parts, joiner)

	// Add a top-level comment for yaml
	if p.currentValue.Comment() != "" && len(p.path) == 0 {
		output = "# " + p.currentValue.Comment() + "\n" + output
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

		if p.Style == JSON {
			parts[i] = p.indent(fmtValue)
		} else {
			parts[i] = fmt.Sprintf("- %s", p.indent(fmtValue))
		}

		if p.currentValue.Comment() != "" {
			if p.Style == JSON {
				parts[i] += "  // " + p.currentValue.Comment()
			} else {
				parts[i] += "  # " + p.currentValue.Comment()
			}
		}

		p.pop()
	}

	if p.Style == JSON {
		if p.currentValue.Comment() != "" {
			return "[  // " + p.currentValue.Comment() + "\n" + strings.Join(parts, ",\n") + "\n]"
		}

		return "[\n" + strings.Join(parts, ",\n") + "\n]"
	}

	return strings.Join(parts, "\n")
}

func (p encoder) format() string {
	switch v := p.currentValue.Value().(type) {
	case cfn.Template:
		return p.formatMap(v.Map())
	case map[string]interface{}:
		return p.formatMap(v)
	case []interface{}:
		return p.formatList(v)
	case string:
		if p.Style == JSON {
			return fmt.Sprintf("%q", v)
		}

		return formatString(v)
	case float64:
		out := fmt.Sprintf("%f", v)
		out = strings.TrimRight(out, "0")
		out = strings.TrimRight(out, ".")
		return out
	default:
		return fmt.Sprint(v)
	}
}
