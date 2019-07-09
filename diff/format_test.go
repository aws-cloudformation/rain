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
		"+ [0]:\n+   bar: baz\n+   quux: mooz\n",
	},
	{
		// Complex value inside a map
		diffMap{
			"foo": diffValue{[]interface{}{
				"bar",
				"baz",
			}, Changed},
		},
		"| foo:\n+   - bar\n+   - baz\n",
	},
	{
		// Add and remove a value from a slice
		diffSlice{
			diffValue{"foo", Unchanged},
			diffValue{"bar", Changed},
			diffValue{"baz", Unchanged},
			diffValue{"quux", Removed},
			diffValue{"mooz", Unchanged},
		},
		"| [1]: bar\n- [3]: ...\n",
	},
	{
		// Add and remove a value from a map
		diffMap{
			"foo": diffValue{"bar", Changed},
			"bar": diffValue{"baz", Removed},
		},
		"- bar: ...\n| foo: bar\n",
	},
	{
		// A slice in a slice
		diffSlice{
			diffSlice{
				diffValue{"foo", Added},
			},
		},
		"+ [0]:\n+   [0]: foo\n",
	},
	{
		// A map in a slice
		diffSlice{
			diffMap{
				"foo": diffValue{"bar", Changed},
			},
		},
		"| [0]:\n|   foo: bar\n",
	},
	{
		// A map in a map
		diffMap{
			"foo": diffMap{
				"bar": diffValue{"baz", Added},
			},
		},
		"+ foo:\n+   bar: baz\n",
	},
	{
		// A slice in a map
		diffMap{
			"foo": diffSlice{
				diffValue{"bar", Changed},
			},
		},
		"| foo:\n|   [0]: bar\n",
	},
	{
		// All added slice in a slice
		diffSlice{
			diffSlice{
				diffValue{"foo", Changed},
				diffValue{"bar", Added},
			},
		},
		"| [0]:\n|   [0]: foo\n+   [1]: bar\n",
	},
	{
		// Mixed slice in a slice
		diffSlice{
			diffSlice{
				diffValue{"foo", Changed},
				diffValue{"bar", Unchanged},
			},
		},
		"| [0]:\n|   [0]: foo\n",
	},
	{
		// Mixed map in a map
		diffMap{
			"foo": diffMap{
				"bar":  diffValue{"baz", Added},
				"quux": diffValue{"mooz", Unchanged},
			},
		},
		"| foo:\n+   bar: baz\n",
	},
	{
		// A new single-value map
		diffMap{
			"foo": diffValue{map[string]interface{}{
				"bar": "baz",
			}, Added},
		},
		"+ foo:\n+   bar: baz\n",
	},
	{
		// A new single-value list
		diffMap{
			"foo": diffValue{[]interface{}{
				"bar",
			}, Added},
		},
		"+ foo:\n+   - bar\n",
	},
	{
		// A new multi-value map
		diffMap{
			"foo": diffValue{map[string]interface{}{
				"bar":  "baz",
				"quux": "mooz",
			}, Added},
		},
		"+ foo:\n+   bar: baz\n+   quux: mooz\n",
	},
	{
		// A new multi-value list
		diffMap{
			"foo": diffValue{[]interface{}{
				"bar",
				"baz",
			}, Added},
		},
		"+ foo:\n+   - bar\n+   - baz\n",
	},
	{
		diffMap{
			"Resources": diffMap{
				"Bucket1": diffMap{
					"Properties": diffValue{
						value: map[string]interface{}{
							"BucketName": map[string]interface{}{
								"Ref": "BucketName",
							},
						},
						valueMode: "- ",
					},
					"Type": diffValue{
						value:     "AWS::S3::Bucket",
						valueMode: "  ",
					},
				},
				"Bucket2": diffValue{
					value: map[string]interface{}{
						"Properties": map[string]interface{}{
							"BucketName": map[string]interface{}{
								"Ref": "Bucket1",
							},
						},
						"Type": "AWS::S3::Bucket",
					},
					valueMode: "+ ",
				},
			},
		},
		"| Resources:\n|   Bucket1:\n-     Properties: {...}\n+   Bucket2:\n+     Properties:\n+       BucketName: !Ref Bucket1\n+     Type: \"AWS::S3::Bucket\"\n",
	},
}

func TestFormat(t *testing.T) {

	for _, testCase := range testCases {
		actual := Format(testCase.value, false)

		if actual != testCase.expected {
			t.Errorf("\n%s\nDOES NOT MATCH\n\n%s\n", actual, testCase.expected)
		}
	}
}
