package cft_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestComments(t *testing.T) {
	var base = `
foo: bar
baz:
  quux: mooz
  xyzzy:
    - corge
    - grault: garply
waldo: {}
`

	var comments = []*cft.Comment{
		&cft.Comment{Path: []interface{}{}, Value: "Comment about the doc"},
		&cft.Comment{Path: []interface{}{"foo"}, Value: "Comment about foo:bar"},
		&cft.Comment{Path: []interface{}{"baz"}, Value: "Comment about baz"},
		&cft.Comment{Path: []interface{}{"baz", "quux"}, Value: "Comment about quux:mooz"},
		&cft.Comment{Path: []interface{}{"baz", "xyzzy"}, Value: "Comment about xyzzy"},
		&cft.Comment{Path: []interface{}{"baz", "xyzzy", 0}, Value: "Comment about corge"},
		&cft.Comment{Path: []interface{}{"baz", "xyzzy", 1}, Value: "Comment about grault"},
		&cft.Comment{Path: []interface{}{"baz", "xyzzy", 1, "grault"}, Value: "Comment about grault:garply"},
		&cft.Comment{Path: []interface{}{"waldo"}, Value: "Comment about waldo"},
	}

	expected := `# Comment about the doc

foo: bar # Comment about foo:bar

baz: # Comment about baz
  quux: mooz # Comment about quux:mooz

  xyzzy: # Comment about xyzzy
    - corge # Comment about corge
    # Comment about grault
    - grault: garply # Comment about grault:garply

waldo: {} # Comment about waldo`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(base), &node)
	if err != nil {
		panic(err)
	}

	tmpl, _ := parse.Node(node)
	tmpl.AddComments(comments)

	actual := format.String(tmpl, format.Options{})

	if d := cmp.Diff(actual, expected); d != "" {
		t.Errorf(d)
	}
}
