package s11n_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/internal/s11n"
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
		actual, err := s11n.GetPath(&base, testCase.path)
		if err != nil {
			t.Error(err)
		}

		if actual != testCase.expected {
			t.Errorf("%#v\n!=\n%#v\n", actual, testCase.expected)
		}
	}
}

func TestGetMap(t *testing.T) {
	var base yaml.Node
	err := yaml.Unmarshal([]byte(nodeTestBase), &base)
	if err != nil {
		t.Fatal(err)
	}
	n := base.Content[0]
	m := s11n.GetMap(n, "baz")
	if m == nil {
		t.Fatal("expected baz map")
	}
	if v, ok := m["quux"]; ok {
		if v.Value != "mooz" {
			t.Fatalf("expected baaz.quux to be mooz")
		}
	} else {
		t.Fatalf("expected to find baaz.quux")
	}
}

func TestGetValue(t *testing.T) {
	var base yaml.Node
	err := yaml.Unmarshal([]byte(nodeTestBase), &base)
	if err != nil {
		t.Fatal(err)
	}
	n := base.Content[0]
	v := s11n.GetValue(n, "foo")
	if v != "bar" {
		t.Fatal("expected foo: bar")
	}
}
