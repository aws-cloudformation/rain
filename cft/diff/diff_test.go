package diff

import (
	"reflect"
	"testing"
)

type compareTest struct {
	old      interface{}
	new      interface{}
	expected string
	mode     Mode
}

func testCompare(t *testing.T, cases []compareTest) {
	for _, testCase := range cases {
		actual := compareValues(testCase.old, testCase.new)

		if actual.Mode() != testCase.mode {
			t.Errorf("Unexpected mode: '%s' (want '%s')", actual.Mode(), testCase.mode)
		}

		if !reflect.DeepEqual(actual.String(), testCase.expected) {
			t.Errorf("%#v\n!=\n%#v", actual.String(), testCase.expected)
		}
	}
}

func TestCompareScalar(t *testing.T) {
	testCompare(t, []compareTest{
		{
			nil, nil, "(=)<nil>", Unchanged,
		},
		{
			"foo", "foo", "(=)foo", Unchanged,
		},
		{
			"foo", "bar", "(>)bar", Changed,
		},
		{
			"foo", 1, "(>)1", Changed,
		},
		{
			"foo", []int{1, 2, 3}, "(>)[1 2 3]", Changed,
		},
	})
}

func TestCompareSlices(t *testing.T) {
	testCompare(t, []compareTest{
		{
			[]interface{}{},
			[]interface{}{},
			"(=)[]",
			Unchanged,
		},
		{
			[]interface{}{1, 2, 3},
			[]interface{}{1, 2, 3},
			"(=)[(=)1 (=)2 (=)3]",
			Unchanged,
		},
		{
			[]interface{}{1, 2, 3},
			[]interface{}{1, 2, 4},
			"(|)[(=)1 (=)2 (>)4]",
			Involved,
		},
		{
			[]interface{}{1, 2, 3},
			[]interface{}{1, 2, 3, 4},
			"(|)[(=)1 (=)2 (=)3 (+)4]",
			Involved,
		},
		{
			[]interface{}{1, 2, 3},
			[]interface{}{1, 2},
			"(|)[(=)1 (=)2 (-)3]",
			Involved,
		},
	})
}

func TestCompareMaps(t *testing.T) {
	testCompare(t, []compareTest{
		{
			map[string]interface{}{},
			map[string]interface{}{},
			"(=)map[]",
			Unchanged,
		},
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{"foo": "bar"},
			"(=)map[foo:(=)bar]",
			Unchanged,
		},
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{"foo": "baz"},
			"(|)map[foo:(>)baz]",
			Involved,
		},
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{"foo": "bar", "baz": "quux"},
			"(|)map[baz:(+)quux foo:(=)bar]",
			Involved,
		},
		{
			map[string]interface{}{"foo": "bar"},
			map[string]interface{}{},
			"(|)map[foo:(-)bar]",
			Involved,
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
			"(=)map[foo:(=)[(=)map[baz:(=)[(=)foo (=)bar] foo:(=)bar] (=)foo]]",
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
			"(|)map[bar:(+)baz foo:(|)[(|)map[baz:(|)[(=)foo (=)bar (+)baz] foo:(=)bar quux:(+)mooz] (=)foo (+)bar]]",
			Involved,
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
			"(|)map[foo:(|)[(|)map[baz:(|)[(=)foo (-)bar] foo:(-)bar] (-)foo]]",
			Involved,
		},
	})
}
