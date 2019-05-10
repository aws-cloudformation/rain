package diff

import (
	"testing"
)

func TestDiffMode(t *testing.T) {
	cases := []struct {
		diff     diff
		expected mode
	}{
		{
			diffValue{"foo", added},
			added,
		},
		{
			diffValue{"foo", changed},
			changed,
		},
		{
			diffSlice{
				added,
			},
			added,
		},
		{
			diffSlice{
				added,
				added,
			},
			added,
		},
		{
			diffSlice{
				removed,
				removed,
			},
			removed,
		},
		{
			diffSlice{
				added,
				removed,
			},
			changed,
		},
		{
			diffMap{
				"foo": added,
			},
			added,
		},
		{
			diffMap{
				"foo": added,
				"bar": added,
			},
			added,
		},
		{
			diffMap{
				"foo": removed,
				"bar": removed,
			},
			removed,
		},
		{
			diffMap{
				"foo": added,
				"bar": removed,
			},
			changed,
		},
	}

	for _, testCase := range cases {
		actual := testCase.diff.mode()

		if actual != testCase.expected {
			t.Errorf("%#v\n!=\n%#v", actual, testCase.expected)
		}
	}
}
