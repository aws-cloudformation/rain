package diff

import (
	"testing"
)

var diffCases = []struct {
	value         Diff
	expectedShort string
	expectedLong  string
}{
	{
		// A single value
		compareValues("foo", "bar"),
		"bar",
		"bar",
	},
	{
		// A more complex single value
		compareValues("foo", map[string]interface{}{
			"foo": "bar",
			"baz": "quux",
		}),
		"baz: quux\nfoo: bar",
		"baz: quux\nfoo: bar",
	},
	{
		// Complex value inside a slice
		compareValues(
			[]interface{}{},
			[]interface{}{
				map[string]interface{}{
					"bar":  "baz",
					"quux": "mooz",
				},
			},
		),
		"(+) [0]:\n(+)   bar: baz\n(+)   quux: mooz\n",
		"(+) [0]:\n(+)   bar: baz\n(+)   quux: mooz\n",
	},
	{
		// Complex value inside a map
		compareValues(
			map[string]interface{}{},
			map[string]interface{}{
				"foo": []interface{}{
					"bar",
					"baz",
				},
			},
		),
		"(+) foo:\n(+)   - bar\n(+)   - baz\n",
		"(+) foo:\n(+)   - bar\n(+)   - baz\n",
	},
	{
		// Add and remove a value from a slice
		compareValues(
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
		"(>) [1]: cake\n(-) [2]: ...\n",
		"(=) [0]: foo\n(>) [1]: cake\n(-) [2]: baz\n",
	},
	{
		// Add and remove a value from a map
		compareValues(
			map[string]interface{}{
				"foo": "bar",
				"baz": "quux",
			},
			map[string]interface{}{
				"foo": "cake",
			},
		),
		"(-) baz: ...\n(>) foo: cake\n",
		"(-) baz: quux\n(>) foo: cake\n",
	},
	{
		// Remove a whole slice
		compareValues(
			map[string]interface{}{
				"foo": []interface{}{"bar"},
			},
			map[string]interface{}{},
		),
		"(-) foo: [...]\n",
		"(-) foo:\n(-)   - bar\n",
	},
	{
		// Remove a whole map
		compareValues(
			map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "baz",
				},
			},
			map[string]interface{}{},
		),
		"(-) foo: {...}\n",
		"(-) foo:\n(-)   bar: baz\n",
	},
}

func TestDiff(t *testing.T) {
	for _, testCase := range diffCases {
		actualShort := testCase.value.Format(false)
		if actualShort != testCase.expectedShort {
			t.Errorf("\n%s\nDOES NOT MATCH SHORT\n%s\n", actualShort, testCase.expectedShort)
		}

		actualLong := testCase.value.Format(true)
		if actualLong != testCase.expectedLong {
			t.Errorf("\n%s\nDOES NOT MATCH LONG\n%s\n", actualLong, testCase.expectedLong)
		}
	}
}
