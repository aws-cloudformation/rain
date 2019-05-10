package diff

import (
	"reflect"
	"testing"
)

func TestCompareScalar(t *testing.T) {
	cases := []struct {
		old      interface{}
		new      interface{}
		expected diff
	}{
		{
			"foo", "foo", unchanged,
		},
		{
			"foo", "bar", diffValue{"bar", changed},
		},
		{
			"foo", 1, diffValue{1, changed},
		},
		{
			"foo", []int{1, 2, 3}, diffValue{[]int{1, 2, 3}, changed},
		},
	}

	for _, testCase := range cases {
		actual := Compare(testCase.old, testCase.new)

		if !reflect.DeepEqual(actual, testCase.expected) {
			t.Errorf("%#v\n!=\n%#v", actual, testCase.expected)
		}
	}
}

func TestCompareSlices(t *testing.T) {
	cases := []struct {
		old      []interface{}
		new      []interface{}
		expected diff
	}{
		{
			[]interface{}{1, 2, 3}, []interface{}{1, 2, 3}, unchanged,
		},
		{
			[]interface{}{1, 2, 3}, []interface{}{1, 2, 4}, diffSlice{
				unchanged,
				unchanged,
				diffValue{4, changed},
			},
		},
		{
			[]interface{}{1, 2, 3}, []interface{}{1, 2, 3, 4}, diffSlice{
				unchanged,
				unchanged,
				unchanged,
				diffValue{4, added},
			},
		},
		{
			[]interface{}{1, 2, 3}, []interface{}{1, 2}, diffSlice{
				unchanged,
				unchanged,
				removed,
			},
		},
	}

	for _, testCase := range cases {
		actual := Compare(testCase.old, testCase.new)

		if !reflect.DeepEqual(actual, testCase.expected) {
			t.Errorf("%#v\n!=\n%#v", actual, testCase.expected)
		}
	}
}

func TestCompareMaps(t *testing.T) {
	cases := []struct {
		old      map[string]interface{}
		new      map[string]interface{}
		expected diff
	}{
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{"foo": "bar"},
			unchanged,
		},
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{"foo": "baz"},
			diffMap{"foo": diffValue{"baz", changed}},
		},
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{"foo": "bar", "baz": "quux"},
			diffMap{"foo": unchanged, "baz": diffValue{"quux", added}},
		},
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{},
			diffMap{"foo": removed},
		},
	}

	for _, testCase := range cases {
		actual := Compare(testCase.old, testCase.new)

		if !reflect.DeepEqual(actual, testCase.expected) {
			t.Errorf("%#v\n!=\n%#v", actual, testCase.expected)
		}
	}
}

func TestCompare(t *testing.T) {
	original := map[string]interface{}{
		"foo": []interface{}{
			map[string]interface{}{
				"foo": "bar",
				"baz": []interface{}{
					"foo",
					"bar",
				},
			},
			"foo",
		},
	}

	cases := []struct {
		old      interface{}
		new      interface{}
		expected diff
	}{
		{
			original,
			original,
			unchanged,
		},
		{
			original,
			map[string]interface{}{
				"foo": []interface{}{
					map[string]interface{}{
						"foo": "bar",
						"baz": []interface{}{
							"foo",
							"bar",
							"baz",
						},
						"quux": "mooz",
					},
					"foo",
					"bar",
				},
				"bar": "baz",
			},
			diffMap{
				"foo": diffSlice{
					diffMap{
						"foo": unchanged,
						"baz": diffSlice{
							unchanged,
							unchanged,
							diffValue{"baz", added},
						},
						"quux": diffValue{"mooz", added},
					},
					unchanged,
					diffValue{"bar", added},
				},
				"bar": diffValue{"baz", added},
			},
		},
		{
			original,
			map[string]interface{}{
				"foo": []interface{}{
					map[string]interface{}{
						"baz": []interface{}{
							"foo",
						},
					},
				},
			},
			diffMap{
				"foo": diffSlice{
					diffMap{
						"foo": removed,
						"baz": diffSlice{
							unchanged,
							removed,
						},
					},
					removed,
				},
			},
		},
	}

	for _, testCase := range cases {
		actual := Compare(testCase.old, testCase.new)

		if !reflect.DeepEqual(actual, testCase.expected) {
			t.Errorf("%#v\n!=\n%#v", actual, testCase.expected)
		}
	}
}
