package diff

import (
	"reflect"
	"testing"
)

func TestCompareScalar(t *testing.T) {
	cases := []struct {
		old      interface{}
		new      interface{}
		expected Diff
	}{
		{
			"foo", "foo", Unchanged,
		},
		{
			"foo", "bar", diffValue{"bar", Changed},
		},
		{
			"foo", 1, diffValue{1, Changed},
		},
		{
			"foo", []int{1, 2, 3}, diffValue{[]int{1, 2, 3}, Changed},
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
		expected Diff
	}{
		{
			[]interface{}{1, 2, 3}, []interface{}{1, 2, 3}, Unchanged,
		},
		{
			[]interface{}{1, 2, 3}, []interface{}{1, 2, 4}, diffSlice{
				Unchanged,
				Unchanged,
				diffValue{4, Changed},
			},
		},
		{
			[]interface{}{1, 2, 3}, []interface{}{1, 2, 3, 4}, diffSlice{
				Unchanged,
				Unchanged,
				Unchanged,
				diffValue{4, Added},
			},
		},
		{
			[]interface{}{1, 2, 3}, []interface{}{1, 2}, diffSlice{
				Unchanged,
				Unchanged,
				Removed,
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
		expected Diff
	}{
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{"foo": "bar"},
			Unchanged,
		},
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{"foo": "baz"},
			diffMap{"foo": diffValue{"baz", Changed}},
		},
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{"foo": "bar", "baz": "quux"},
			diffMap{"foo": Unchanged, "baz": diffValue{"quux", Added}},
		},
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{},
			diffMap{"foo": Removed},
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
		expected Diff
	}{
		{
			original,
			original,
			Unchanged,
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
						"foo": Unchanged,
						"baz": diffSlice{
							Unchanged,
							Unchanged,
							diffValue{"baz", Added},
						},
						"quux": diffValue{"mooz", Added},
					},
					Unchanged,
					diffValue{"bar", Added},
				},
				"bar": diffValue{"baz", Added},
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
						"foo": Removed,
						"baz": diffSlice{
							Unchanged,
							Removed,
						},
					},
					Removed,
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
