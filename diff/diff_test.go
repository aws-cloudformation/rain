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
				Added,
			},
			Added,
		},
		{
			diffSlice{
				Added,
				Added,
			},
			Added,
		},
		{
			diffSlice{
				Removed,
				Removed,
			},
			Removed,
		},
		{
			diffSlice{
				Added,
				Removed,
			},
			Changed,
		},
		{
			diffMap{
				"foo": Added,
			},
			Added,
		},
		{
			diffMap{
				"foo": Added,
				"bar": Added,
			},
			Added,
		},
		{
			diffMap{
				"foo": Removed,
				"bar": Removed,
			},
			Removed,
		},
		{
			diffMap{
				"foo": Added,
				"bar": Removed,
			},
			Changed,
		},
	}

	for _, testCase := range cases {
		actual := testCase.value.mode()

		if actual != testCase.expected {
			t.Errorf("%#v\n!=\n%#v", actual, testCase.expected)
		}
	}
}
