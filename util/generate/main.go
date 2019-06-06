package main

import (
	"fmt"
	"io/ioutil"
	"strings"
)

var colours = map[string]string{
	"Bold":   "\033[1;37m",
	"Orange": "\033[0;33m",
	"Yellow": "\033[1;33m",
	"Red":    "\033[1;31m",
	"Green":  "\033[0;32m",
	"Grey":   "\033[0;37m",
	"White":  "\033[1;37m",
}

func main() {
	output := strings.Builder{}

	output.WriteString("package util\n")

	for name, code := range colours {
		output.WriteString(fmt.Sprintf(`
func %s (text string) Text {
    return Text{
        text: text,
        colour: %q,
    }
}
`, name, code))
	}

	ioutil.WriteFile("colours.go", []byte(output.String()), 0644)
}
