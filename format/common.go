package format

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

func sortKeys(data map[string]interface{}, path []interface{}) []string {
	// See if we have some required-order keys
	order := ordering(path)

	done := make(map[string]bool)
	orderedKeys := make([]string, 0, len(data))
	otherKeys := make([]string, 0, len(data))

	// Apply required keys
	for _, orderedKey := range order {
		if _, ok := data[orderedKey]; ok {
			orderedKeys = append(orderedKeys, orderedKey)
			done[orderedKey] = true
		}
	}

	// Now the remainder of the keys
	for key := range data {
		if !done[key] {
			otherKeys = append(otherKeys, key)
		}
	}
	sort.Strings(otherKeys)

	return append(orderedKeys, otherKeys...)
}

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
	quote := false

	switch {
	case strings.ContainsAny(data, "\n"),
		data == "",
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
		quote = true
	}

	if quote {
		return fmt.Sprintf("%q", data)
	}

	return data
}
