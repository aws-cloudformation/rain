package diff

import (
	"testing"
)

var testCases = []struct {
	value    Diff
	expected string
}{
	{
		// A single value
		diffValue{"cake", Added},
		"cake",
	},
	{
		// A more complex single value
		diffValue{map[string]interface{}{
			"foo": "bar",
			"baz": "quux",
		}, Added},
		"baz: quux\nfoo: bar",
	},
	{
		// Complex value inside a slice
		diffSlice{
			diffValue{map[string]interface{}{
				"bar":  "baz",
				"quux": "mooz",
			}, Added},
		},
		">>> [0]:\n>>>   bar: baz\n>>>   quux: mooz\n",
	},
	{
		// Complex value inside a map
		diffMap{
			"foo": diffValue{[]interface{}{
				"bar",
				"baz",
			}, Changed},
		},
		">>> foo:\n>>>   - bar\n>>>   - baz\n",
	},
	{
		// Add and remove a value from a slice
		diffSlice{
			Unchanged,
			diffValue{"foo", Changed},
			Unchanged,
			Removed,
			Unchanged,
		},
		">>> [1]: foo\n<<< [3]\n",
	},
	{
		// Add and remove a value from a map
		diffMap{
			"foo": diffValue{"bar", Changed},
			"bar": Removed,
		},
		"<<< bar\n>>> foo: bar\n",
	},
	{
		// A slice in a slice
		diffSlice{
			diffSlice{
				diffValue{"foo", Added},
			},
		},
		">>> [0]:\n>>>   [0]: foo\n",
	},
	{
		// A map in a slice
		diffSlice{
			diffMap{
				"foo": diffValue{"bar", Changed},
			},
		},
		"||| [0]:\n>>>   foo: bar\n",
	},
	{
		// A map in a map
		diffMap{
			"foo": diffMap{
				"bar": diffValue{"baz", Added},
			},
		},
		">>> foo:\n>>>   bar: baz\n",
	},
	{
		// A slice in a map
		diffMap{
			"foo": diffSlice{
				diffValue{"bar", Changed},
			},
		},
		"||| foo:\n>>>   [0]: bar\n",
	},
	{
		// All added slice in a slice
		diffSlice{
			diffSlice{
				diffValue{"foo", Changed},
				diffValue{"bar", Added},
			},
		},
		"||| [0]:\n>>>   [0]: foo\n>>>   [1]: bar\n",
	},
	{
		// Mixed slice in a slice
		diffSlice{
			diffSlice{
				diffValue{"foo", Changed},
				Unchanged,
			},
		},
		"||| [0]:\n>>>   [0]: foo\n",
	},
	{
		// Mixed map in a map
		diffMap{
			"foo": diffMap{
				"bar":  diffValue{"baz", Added},
				"quux": Unchanged,
			},
		},
		"||| foo:\n>>>   bar: baz\n",
	},
}

func TestFormat(t *testing.T) {

	for _, testCase := range testCases {
		actual := Format(testCase.value)

		if actual != testCase.expected {
			t.Errorf("%q\n!=\n%q", actual, testCase.expected)
		}
	}
}
