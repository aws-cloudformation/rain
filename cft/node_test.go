package cft

import (
	"testing"

	"gopkg.in/yaml.v3"
)

var nodeTestBase = `
foo: bar
baz:
  quux: mooz
  xyzzy:
    - corge
    - grault: garply
`

func TestGetNodePath(t *testing.T) {
	var base yaml.Node
	err := yaml.Unmarshal([]byte(nodeTestBase), &base)
	if err != nil {
		panic(err)
	}

	testCases := []struct {
		path     []interface{}
		expected interface{}
	}{
		{[]interface{}{}, base.Content[0]},
		{[]interface{}{"foo"}, base.Content[0].Content[1]},
		{[]interface{}{"baz"}, base.Content[0].Content[3]},
		{[]interface{}{"baz", "quux"}, base.Content[0].Content[3].Content[1]},
		{[]interface{}{"baz", "xyzzy"}, base.Content[0].Content[3].Content[3]},
		{[]interface{}{"baz", "xyzzy", 0}, base.Content[0].Content[3].Content[3].Content[0]},
		{[]interface{}{"baz", "xyzzy", 1}, base.Content[0].Content[3].Content[3].Content[1]},
		{[]interface{}{"baz", "xyzzy", 1, "grault"}, base.Content[0].Content[3].Content[3].Content[1].Content[1]},
	}

	for _, testCase := range testCases {
		actual, err := getNodePath(&base, testCase.path)
		if err != nil {
			t.Error(err)
		}

		if actual != testCase.expected {
			t.Errorf("%#v\n!=\n%#v\n", actual, testCase.expected)
		}
	}
}
