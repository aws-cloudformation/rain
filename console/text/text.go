// Package text defines the Text type that is used to represent blocks of
// ANSI formatted text. The package also defines generated helper functions
// for creating different coloured blocks of text.
package text

import (
	"fmt"

	"github.com/aws-cloudformation/rain/console"
)

//go:generate go run generate/main.go

const end = "\033[0m"

// A Text represents a string and a colour.
type Text struct {
	text   string
	colour string
}

// String returns a formatted (if supported) string of the Text
func (t Text) String() string {
	if t.colour == "" || !console.IsTTY || !console.HasColour {
		return t.Plain()
	}

	return fmt.Sprintf("%s%s%s", t.colour, t.text, end)
}

// Len returns the length of the text
func (t Text) Len() int {
	return len(t.text)
}

// Plain returns the unformatted text
func (t Text) Plain() string {
	return t.text
}
