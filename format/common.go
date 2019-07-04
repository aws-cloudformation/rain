package format

import (
	"fmt"
	"reflect"
	"strings"
)

func intrinsicKey(data map[string]interface{}) (string, bool) {
	if len(data) != 1 {
		return "", false
	}

	// We know there's one key
	key := reflect.ValueOf(data).MapKeys()[0].String()
	if key == "Ref" || strings.HasPrefix(key, "Fn::") {
		return key, true
	}

	return "", false
}

func formatString(data string) string {
	switch {
	case strings.ContainsAny(data, "\n"):
		parts := strings.Split(strings.TrimSpace(data), "\n")
		for i, part := range parts {
			parts[i] = "  " + part
		}
		return fmt.Sprintf("|\n%s", strings.Join(parts, "\n"))
	case data == "",
		strings.ToLower(data) == "yes",
		strings.ToLower(data) == "no",
		strings.ToLower(data) == "y",
		strings.ToLower(data) == "n",
		strings.ToLower(data) == "true",
		strings.ToLower(data) == "false",
		strings.ToLower(data) == "null",
		strings.ContainsAny(string(data[0]), "0123456789!&%*?,|>@[{}]-\\ \t\n"),
		strings.ContainsAny(string(data[len(data)-1]), " \t\n"),
		strings.ContainsAny(data, "`\"':#"):
		return fmt.Sprintf("%q", data)
	}

	return data
}
