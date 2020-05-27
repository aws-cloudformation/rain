package format_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/aws-cloudformation/rain/cfn/value"
)

func TestJsonScalars(t *testing.T) {
	cases := []map[string]interface{}{
		{"foo": 1},
		{"foo": 1.0},
		{"foo": 1.234},
		{"foo": "32"},
		{"foo": "032"},
		{"foo": "32.0"},
		{"foo": "hello"},
		{"foo": true},
		{"foo": false},
	}

	expecteds := []string{
		"{\n    \"foo\": 1\n}",
		"{\n    \"foo\": 1\n}",
		"{\n    \"foo\": 1.234\n}",
		"{\n    \"foo\": \"32\"\n}",
		"{\n    \"foo\": \"032\"\n}",
		"{\n    \"foo\": \"32.0\"\n}",
		"{\n    \"foo\": \"hello\"\n}",
		"{\n    \"foo\": true\n}",
		"{\n    \"foo\": false\n}",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual := format.Anything(testCase, format.Options{
			Style: format.JSON,
		})

		if actual != expected {
			t.Errorf("from %T %v:\n%#v != %#v\n", testCase, testCase, actual, expected)
		}
	}
}

func TestJsonList(t *testing.T) {
	cases := []map[string]interface{}{
		{"foo": []interface{}{}},
		{"foo": []interface{}{1}},
		{"foo": []interface{}{
			1,
			"foo",
			true,
		}},
		{"foo": []interface{}{
			[]interface{}{
				"foo",
				"bar",
			},
			"baz",
		}},
		{"foo": []interface{}{
			[]interface{}{
				[]interface{}{
					"foo",
					"bar",
				},
				"baz",
			},
			"quux",
		}},
		{"foo": []interface{}{
			map[string]interface{}{
				"foo": "bar",
			},
			map[string]interface{}{
				"baz":  "quux",
				"mooz": "xyzzy",
			},
		}},
	}

	expecteds := []string{
		"{\n    \"foo\": []\n}",
		"{\n    \"foo\": [\n        1\n    ]\n}",
		"{\n    \"foo\": [\n        1,\n        \"foo\",\n        true\n    ]\n}",
		"{\n    \"foo\": [\n        [\n            \"foo\",\n            \"bar\"\n        ],\n        \"baz\"\n    ]\n}",
		"{\n    \"foo\": [\n        [\n            [\n                \"foo\",\n                \"bar\"\n            ],\n            \"baz\"\n        ],\n        \"quux\"\n    ]\n}",
		"{\n    \"foo\": [\n        {\n            \"foo\": \"bar\"\n        },\n        {\n            \"baz\": \"quux\",\n            \"mooz\": \"xyzzy\"\n        }\n    ]\n}",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual := format.Anything(testCase, format.Options{
			Style: format.JSON,
		})

		if actual != expected {
			t.Errorf("\n%v\n  is not\n%v\n", actual, expected)
		}
	}
}

func TestJsonMap(t *testing.T) {
	cases := []map[string]interface{}{
		{},
		{
			"foo": "bar",
		},
		{
			"foo": "bar",
			"baz": "quux",
		},
		{
			"foo": map[string]interface{}{
				"bar": "baz",
			},
			"quux": "mooz",
		},
		{
			"foo": map[string]interface{}{
				"bar": map[string]interface{}{
					"baz": "quux",
				},
				"mooz": "xyzzy",
			},
			"alpha": "beta",
		},
		{
			"foo": []interface{}{
				"bar",
				"baz",
			},
			"quux": []interface{}{
				"mooz",
			},
		},
	}

	expecteds := []string{
		"{}",
		"{\n    \"foo\": \"bar\"\n}",
		"{\n    \"baz\": \"quux\",\n\n    \"foo\": \"bar\"\n}",
		"{\n    \"foo\": {\n        \"bar\": \"baz\"\n    },\n\n    \"quux\": \"mooz\"\n}",
		"{\n    \"alpha\": \"beta\",\n\n    \"foo\": {\n        \"bar\": {\n            \"baz\": \"quux\"\n        },\n\n        \"mooz\": \"xyzzy\"\n    }\n}",
		"{\n    \"foo\": [\n        \"bar\",\n        \"baz\"\n    ],\n\n    \"quux\": [\n        \"mooz\"\n    ]\n}",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual := format.Anything(testCase, format.Options{
			Style: format.JSON,
		})

		if actual != expected {
			t.Errorf("\n%v\n---IS NOT---\n%v\n", actual, expected)
		}
	}
}

func TestCfnJson(t *testing.T) {
	cases := []map[string]interface{}{
		{
			"Quux":       "mooz",
			"Parameters": "baz",
			"Foo":        "bar",
			"Resources":  "xyzzy",
		},
	}

	expecteds := []string{
		"{\n    \"Parameters\": \"baz\",\n\n    \"Resources\": \"xyzzy\",\n\n    \"Foo\": \"bar\",\n\n    \"Quux\": \"mooz\"\n}",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual := format.Anything(testCase, format.Options{
			Style: format.JSON,
		})

		if actual != expected {
			t.Errorf("from %T %v:\n%#v != %#v\n", testCase, testCase, actual, expected)
		}
	}
}

func TestJsonComments(t *testing.T) {
	data := map[string]interface{}{
		"foo": "bar",
		"baz": map[string]interface{}{
			"quux": "mooz",
		},
		"xyzzy": []interface{}{
			"lorem",
		},
	}

	comments := []struct {
		path  []interface{}
		value string
	}{
		{[]interface{}{}, "Top-level comment"},
		{[]interface{}{"foo"}, "This is foo"},
		{[]interface{}{"baz"}, "This is baz"},
		{[]interface{}{"baz", "quux"}, "This is quux"},
		{[]interface{}{"xyzzy"}, "This is xyzzy"},
		{[]interface{}{"xyzzy", 0}, "This is lorem"},
	}

	expecteds := []string{
		"{  // Top-level comment\n    \"baz\": {\n        \"quux\": \"mooz\"\n    },\n\n    \"foo\": \"bar\",\n\n    \"xyzzy\": [\n        \"lorem\"\n    ]\n}",
		"{\n    \"baz\": {\n        \"quux\": \"mooz\"\n    },\n\n    \"foo\": \"bar\",  // This is foo\n\n    \"xyzzy\": [\n        \"lorem\"\n    ]\n}",
		"{\n    \"baz\": {  // This is baz\n        \"quux\": \"mooz\"\n    },\n\n    \"foo\": \"bar\",\n\n    \"xyzzy\": [\n        \"lorem\"\n    ]\n}",
		"{\n    \"baz\": {\n        \"quux\": \"mooz\"  // This is quux\n    },\n\n    \"foo\": \"bar\",\n\n    \"xyzzy\": [\n        \"lorem\"\n    ]\n}",
		"{\n    \"baz\": {\n        \"quux\": \"mooz\"\n    },\n\n    \"foo\": \"bar\",\n\n    \"xyzzy\": [  // This is xyzzy\n        \"lorem\"\n    ]\n}",
		"{\n    \"baz\": {\n        \"quux\": \"mooz\"\n    },\n\n    \"foo\": \"bar\",\n\n    \"xyzzy\": [\n        \"lorem\"  // This is lorem\n    ]\n}",
	}

	for i, comment := range comments {
		expected := expecteds[i]

		v := value.New(data)
		v.Get(comment.path...).SetComment(comment.value)

		actual := format.Value(v, format.Options{
			Style: format.JSON,
		})

		if actual != expected {
			t.Errorf("from %q != %q\n", actual, expected)
		}
	}
}
