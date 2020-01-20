package colourise

import (
	"strings"

	"github.com/aws-cloudformation/rain/cfn/diff"
	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/aws-cloudformation/rain/console/text"
)

func Diff(d diff.Diff, longFormat bool) string {
	output := strings.Builder{}

	parts := strings.Split(format.Diff(d, format.Options{Compact: !longFormat}), "\n")

	for i, line := range parts {
		switch {
		case strings.HasPrefix(line, diff.Added.String()):
			output.WriteString(text.Green(line).String())
		case strings.HasPrefix(line, diff.Removed.String()):
			output.WriteString(text.Red(line).String())
		case strings.HasPrefix(line, diff.Changed.String()):
			output.WriteString(text.Orange(line).String())
		case strings.HasPrefix(line, diff.Involved.String()):
			output.WriteString(text.Grey(line).String())
		default:
			output.WriteString(line)
		}

		if i < len(parts)-1 {
			output.WriteString("\n")
		}
	}

	return output.String()
}
