package format_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/cfn/diff"
	"github.com/aws-cloudformation/rain/cfn/format"
)

var diffCases = []struct {
	value    diff.Diff
	expected string
}{
	{
		// A single value
		diff.New("foo", "bar"),
		"bar",
	},
	{
		// A more complex single value
		diff.New("foo", map[string]interface{}{
			"foo": "bar",
			"baz": "quux",
		}),
		"baz: quux\nfoo: bar",
	},
	{
		// Complex value inside a slice
		diff.New(
			[]interface{}{},
			[]interface{}{
				map[string]interface{}{
					"bar":  "baz",
					"quux": "mooz",
				},
			},
		),
		"(+) [0]:\n(+)   bar: baz\n(+)   quux: mooz\n",
	},
	{
		// Complex value inside a map
		diff.New(
			map[string]interface{}{},
			map[string]interface{}{
				"foo": []interface{}{
					"bar",
					"baz",
				},
			},
		),
		"(+) foo:\n(+)   - bar\n(+)   - baz\n",
	},
	{
		// Add and remove a value from a slice
		diff.New(
			[]interface{}{
				"foo",
				"bar",
				"baz",
			},
			[]interface{}{
				"foo",
				"cake",
			},
		),
		"(|) [1]: cake\n(-) [2]: ...\n",
	},
	{
		// Add and remove a value from a map
		diff.New(
			map[string]interface{}{
				"foo": "bar",
				"baz": "quux",
			},
			map[string]interface{}{
				"foo": "cake",
			},
		),
		"(-) baz: ...\n(|) foo: cake\n",
	},
}

func TestDiff(t *testing.T) {
	for _, testCase := range diffCases {
		actual := format.Diff(testCase.value, format.Options{Compact: true})

		if actual != testCase.expected {
			t.Errorf("\n%s\nDOES NOT MATCH\n%s\n", actual, testCase.expected)
		}
	}
}
