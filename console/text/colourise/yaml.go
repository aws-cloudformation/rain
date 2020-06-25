package colourise

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/console/text"
)

// Yaml colourises a Yaml string
func Yaml(in string) string {
	parts := strings.Split(in, "\n")

	for i := range parts {
		line := strings.Split(parts[i], "  # ")
		if len(line) == 2 {
			parts[i] = fmt.Sprintf("%s  %s", line[0], text.Grey("# "+line[1]))
		}

		line = strings.Split(parts[i], ": ")
		if len(line) == 2 {
			parts[i] = fmt.Sprintf("%s: %s", line[0], text.Yellow(line[1]))
		}
	}

	return strings.Join(parts, "\n")
}
