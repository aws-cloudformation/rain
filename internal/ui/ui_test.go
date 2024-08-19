package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/console"
)

func TestColouriseStatus(t *testing.T) {
	for input, colour := range map[string]func(...interface{}) string{
		"ROLLBACK_FAILED":       console.Red,
		"SOMETHING_ELSE_FAILED": console.Red,
		"ROLLBACK_SUCCEEDED":    console.Red,
		"SOMETHING_ROLLBACK":    console.Red,
		"BANANA_IN_PROGRESS":    console.Blue,
		"SOMETHING_COMPLETE":    console.Green,
		"ANOTHER THING":         console.Plain,
	} {
		actual := ColouriseStatus(input)
		expected := colour(input)

		if actual != expected {
			fmt.Printf("Got '%s'. Want: '%s'.\n", actual, expected)
			t.Fail()
		}
	}
}

func TestColouriseDiff(t *testing.T) {
	a, _ := parse.File("../../test/templates/success.template")
	b, _ := parse.File("../../test/templates/failure.template")

	d := diff.New(a, b)

	actual := ColouriseDiff(d, true)

	expected := strings.Join([]string{
		console.Blue("(>) Description: This template fails"),
		console.Red("(-) Parameters:"),
		console.Red("(-)   BucketName:"),
		console.Red("(-)     Type: String"),
		console.Grey("(|) Resources:"),
		console.Grey("(|)   Bucket1:"),
		console.Red("(-)     Properties:"),
		console.Red("(-)       BucketName:"),
		console.Red("(-)         Ref: BucketName"),
		console.Plain("(=)     Type: AWS::S3::Bucket"),
		console.Green("(+)   Bucket2:"),
		console.Green("(+)     Properties:"),
		console.Green("(+)       BucketName:"),
		console.Green("(+)         Ref: Bucket1"),
		console.Green("(+)     Type: AWS::S3::Bucket"),
		console.Plain(""),
	}, "\n")

	if actual != expected {
		fmt.Println("Got:")
		fmt.Println(actual)
		fmt.Println("Want:")
		fmt.Println(expected)
		t.Fail()
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

	actual := Indent("  ", input)

	if d := cmp.Diff(actual, expected); d != "" {
		t.Error(d)
	}
}
