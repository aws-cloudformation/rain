package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/aws-cloudformation/rain/cfn/diff"
	"github.com/aws-cloudformation/rain/console"
	"github.com/aws-cloudformation/rain/console/text"
)

func withColour(f func()) {
	lastTTY := console.IsTTY
	lastColour := console.HasColour

	console.IsTTY = true
	console.HasColour = true

	f()

	console.IsTTY = lastTTY
	console.HasColour = lastColour
}

func TestColouriseStatus(t *testing.T) {
	withColour(func() {
		for input, colour := range map[string]func(string) text.Text{
			"ROLLBACK_FAILED":       text.Red,
			"SOMETHING_ELSE_FAILED": text.Red,
			"ROLLBACK_SUCCEEDED":    text.Orange,
			"SOMETHING_ROLLBACK":    text.Orange,
			"BANANA_IN_PROGRESS":    text.Orange,
			"SOMETHING_COMPLETE":    text.Green,
			"ANOTHER THING":         text.Plain,
		} {
			actual := colouriseStatus(input).String()
			expected := colour(input).Format()

			if actual != expected {
				fmt.Printf("Got '%s'. Want: '%s'.\n", actual, expected)
				t.Fail()
			}
		}
	})
}

func TestColouriseDiff(t *testing.T) {
	withColour(func() {
		actual := colouriseDiff(diff.New(
			map[string]interface{}{
				"foo": []interface{}{
					"bar",
				},
				"baz": "quux",
			},
			map[string]interface{}{
				"foo": []interface{}{
					"baz",
					"quux",
				},
				"baz":  "quux",
				"mooz": "xyzzy",
			},
		), true)

		expected := strings.Join([]string{
			text.Plain("(=) baz: quux").Format(),
			text.Grey("(|) foo:").Format(),
			text.Orange("(>)   [0]: baz").Format(),
			text.Green("(+)   [1]: quux").Format(),
			text.Green("(+) mooz: xyzzy").Format(),
			"",
		}, "\n")

		if actual != expected {
			fmt.Println("Got:")
			fmt.Println(actual)
			fmt.Println("Want:")
			fmt.Println(expected)
			t.Fail()
		}
	})
}

func TestStatusIsSettled(t *testing.T) {
	for input, expected := range map[string]bool{
		"STACK_COMPLETE":     true,
		"STACK_FAILED":       true,
		"SOMETHING_COMPLETE": true,
		"SOMETHING_FAILED":   true,
		"COMPLETE_STACK":     false,
		"FAILED_STACK":       false,
	} {
		if statusIsSettled(input) != expected {
			t.Fail()
		}
	}
}

func TestIndent(t *testing.T) {
	input := `This
has
multiple
lines
`

	expected := `  This
  has
  multiple
  lines` // Should chomp ending blank lines

	actual := indent("  ", input)

	if d := cmp.Diff(actual, expected); d != "" {
		t.Errorf(d)
	}
}
