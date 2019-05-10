package diff

import (
	"testing"
)

var testCases = []struct {
	diff     diff
	expected string
}{
	{
		// A single value
		diffValue{"cake", added},
		"cake",
	},
	{
		// A more complex single value
		diffValue{map[string]interface{}{
			"foo": "bar",
			"baz": "quux",
		}, added},
		"baz: quux\nfoo: bar",
	},
	{
		// Complex value inside a slice
		diffSlice{
			diffValue{map[string]interface{}{
				"bar":  "baz",
				"quux": "mooz",
			}, added},
		},
		">>> [0]:\n>>>   bar: baz\n>>>   quux: mooz\n",
	},
	{
		// Complex value inside a map
		diffMap{
			"foo": diffValue{[]interface{}{
				"bar",
				"baz",
			}, changed},
		},
		">>> foo:\n>>>   - bar\n>>>   - baz\n",
	},
	{
		// Add and remove a value from a slice
		diffSlice{
			unchanged,
			diffValue{"foo", changed},
			unchanged,
			removed,
			unchanged,
		},
		">>> [1]: foo\n<<< [3]\n",
	},
	{
		// Add and remove a value from a map
		diffMap{
			"foo": diffValue{"bar", changed},
			"bar": removed,
		},
		"<<< bar\n>>> foo: bar\n",
	},
	{
		// A slice in a slice
		diffSlice{
			diffSlice{
				diffValue{"foo", added},
			},
		},
		">>> [0]:\n>>>   [0]: foo\n",
	},
	{
		// A map in a slice
		diffSlice{
			diffMap{
				"foo": diffValue{"bar", changed},
			},
		},
		"||| [0]:\n>>>   foo: bar\n",
	},
	{
		// A map in a map
		diffMap{
			"foo": diffMap{
				"bar": diffValue{"baz", added},
			},
		},
		">>> foo:\n>>>   bar: baz\n",
	},
	{
		// A slice in a map
		diffMap{
			"foo": diffSlice{
				diffValue{"bar", changed},
			},
		},
		"||| foo:\n>>>   [0]: bar\n",
	},
	{
		// All added slice in a slice
		diffSlice{
			diffSlice{
				diffValue{"foo", changed},
				diffValue{"bar", added},
			},
		},
		"||| [0]:\n>>>   [0]: foo\n>>>   [1]: bar\n",
	},
	{
		// Mixed slice in a slice
		diffSlice{
			diffSlice{
				diffValue{"foo", changed},
				unchanged,
			},
		},
		"||| [0]:\n>>>   [0]: foo\n",
	},
	{
		// Mixed map in a map
		diffMap{
			"foo": diffMap{
				"bar":  diffValue{"baz", added},
				"quux": unchanged,
			},
		},
		"||| foo:\n>>>   bar: baz\n",
	},
}

func TestFormat(t *testing.T) {

	for _, testCase := range testCases {
		actual := Format(testCase.diff)

		if actual != testCase.expected {
			t.Errorf("%q\n!=\n%q", actual, testCase.expected)
		}
	}
}
