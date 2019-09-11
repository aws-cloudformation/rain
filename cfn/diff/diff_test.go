package diff_test

import (
	"reflect"
	"testing"

	"github.com/aws-cloudformation/rain/cfn/diff"
)

type compareTest struct {
	old   interface{}
	new   interface{}
	value interface{}
	mode  diff.Mode
}

func testCompare(t *testing.T, cases []compareTest) {
	for _, testCase := range cases {
		actual := diff.New(testCase.old, testCase.new)

		if actual.Mode() != testCase.mode {
			t.Errorf("Unexpected mode: '%s' (want '%s')", actual.Mode(), testCase.mode)
		}

		if !reflect.DeepEqual(actual.Value(), testCase.value) {
			t.Errorf("%#v\n!=\n%#v", actual.Value(), testCase.value)
		}
	}
}

func TestCompareScalar(t *testing.T) {
	testCompare(t, []compareTest{
		{
			nil, nil, nil, diff.Unchanged,
		},
		{
			"foo", "foo", "foo", diff.Unchanged,
		},
		{
			"foo", "bar", "bar", diff.Changed,
		},
		{
			"foo", 1, 1, diff.Changed,
		},
		{
			"foo", []int{1, 2, 3}, []int{1, 2, 3}, diff.Changed,
		},
	})
}

func TestCompareSlices(t *testing.T) {
	testCompare(t, []compareTest{
		{
			[]interface{}{},
			[]interface{}{},
			[]interface{}{},
			diff.Unchanged,
		},
		{
			[]interface{}{1, 2, 3},
			[]interface{}{1, 2, 3},
			[]interface{}{1, 2, 3},
			diff.Unchanged,
		},
		{
			[]interface{}{1, 2, 3},
			[]interface{}{1, 2, 4},
			[]interface{}{
				1,
				2,
				4,
			},
			diff.Changed,
		},
		{
			[]interface{}{1, 2, 3},
			[]interface{}{1, 2, 3, 4},
			[]interface{}{
				1,
				2,
				3,
				4,
			},
			diff.Changed,
		},
		{
			[]interface{}{1, 2, 3},
			[]interface{}{1, 2},
			[]interface{}{
				1,
				2,
				3,
			},
			diff.Changed,
		},
	})
}

func TestCompareMaps(t *testing.T) {
	testCompare(t, []compareTest{
		{
			map[string]interface{}{},
			map[string]interface{}{},
			map[string]interface{}{},
			diff.Unchanged,
		},
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{
				"foo": "bar",
			},
			diff.Unchanged,
		},
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{"foo": "baz"},
			map[string]interface{}{
				"foo": "baz",
			},
			diff.Changed,
		},
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{"foo": "bar", "baz": "quux"},
			map[string]interface{}{
				"foo": "bar",
				"baz": "quux",
			},
			diff.Changed,
		},
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{},
			map[string]interface{}{
				"foo": "bar",
			},
			diff.Removed,
		},
	})
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

	testCompare(t, []compareTest{
		{
			original,
			original,
			original,
			diff.Unchanged,
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
			diff.Changed,
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
			map[string]interface{}{
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
			},
			diff.Changed,
		},
	})
}
