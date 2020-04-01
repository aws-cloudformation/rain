package main

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
)

var colours = map[string]string{
	"Blue":   "\033[1;34m",
	"Bold":   "\033[1;37m",
	"Green":  "\033[0;32m",
	"Grey":   "\033[1;30m",
	"Orange": "\033[0;33m",
	"Red":    "\033[1;31m",
	"White":  "\033[1;37m",
	"Yellow": "\033[1;33m",
}

func main() {
	names := make([]string, 0)
	for key := range colours {
		names = append(names, key)
	}
	sort.Strings(names)

	output := strings.Builder{}

	output.WriteString(`package text

// Code generated. DO NOT EDIT.
`)

	for _, name := range names {
		output.WriteString(fmt.Sprintf(`
// %s returns a Text struct that wraps the supplied text in ANSI colour
func %s(text string) Text {
	return Text{
		text:   text,
		colour: %q,
	}
}
`, name, name, colours[name]))
	}

	output.WriteString(fmt.Sprintf(`
// Plain returns a Text struct that always returns the supplied text, unformatted
func Plain(text string) Text {
	return Text{
		text:   text,
		colour: "",
	}
}
`))

	ioutil.WriteFile("colours.go", []byte(output.String()), 0644)
}
