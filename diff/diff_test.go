package diff

import (
	"testing"
)

func TestDiffMode(t *testing.T) {
	cases := []struct {
		value    Diff
		expected mode
	}{
		{
			diffValue{"foo", Added},
			Added,
		},
		{
			diffValue{"foo", Changed},
			Changed,
		},
		{
			diffSlice{
				diffValue{"foo", Added},
			},
			Added,
		},
		{
			diffSlice{
				diffValue{"foo", Added},
				diffValue{"bar", Added},
			},
			Added,
		},
		{
			diffSlice{
				diffValue{"foo", Removed},
				diffValue{"bar", Removed},
			},
			Removed,
		},
		{
			diffSlice{
				diffValue{"foo", Added},
				diffValue{"bar", Removed},
			},
			Changed,
		},
		{
			diffMap{
				"foo": diffValue{"bar", Added},
			},
			Added,
		},
		{
			diffMap{
				"foo": diffValue{"bar", Added},
				"baz": diffValue{"quux", Added},
			},
			Added,
		},
		{
			diffMap{
				"foo": diffValue{"bar", Removed},
				"baz": diffValue{"quux", Removed},
			},
			Removed,
		},
		{
			diffMap{
				"foo": diffValue{"bar", Added},
				"baz": diffValue{"quux", Removed},
			},
			Changed,
		},
	}

	for _, testCase := range cases {
		actual := testCase.value.Mode()

		if actual != testCase.expected {
			t.Errorf("%#v\n!=\n%#v", actual, testCase.expected)
		}
	}
}
