package colourise

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/console/text"
)

func Yaml(in string) string {
	parts := strings.Split(in, "\n")

	for i, part := range parts {
		line := strings.Split(part, ": ")

		if len(line) == 2 {
			parts[i] = fmt.Sprintf("%s:%s", line[0], text.Yellow(line[1]))
		}
	}

	return strings.Join(parts, "\n")
}
