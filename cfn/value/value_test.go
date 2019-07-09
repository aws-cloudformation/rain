package value_test

import (
	"reflect"
	"testing"

	"github.com/aws-cloudformation/rain/cfn/value"
)

var data = map[string]interface{}{
	"foo": "bar",
	"baz": map[string]interface{}{
		"quux": []interface{}{
			"mooz",
		},
		"xyzzy": "lorem",
		"ipsum": "dolor", // Uncommented
	},
}

var comments = map[interface{}]interface{}{
	"":    "Root comment",
	"foo": "This is foo",
	"baz": map[interface{}]interface{}{
		"": "This is baz",
		"quux": map[interface{}]interface{}{
			"": "This is quux",
			0:  "This is quux[0]",
		},
		"xyzzy": "This is xyzzy",
	},
}

var v value.Value = value.New(
	data,
	comments,
)

var paths = [][]interface{}{
	{},
	{"foo"},
	{"baz"},
	{"baz", "quux"},
	{"baz", "quux", 0},
	{"baz", "xyzzy"},
	{"baz", "ipsum"},
}

func TestGet(t *testing.T) {
	expecteds := []interface{}{
		data,
		data["foo"],
		data["baz"],
		data["baz"].(map[string]interface{})["quux"],
		data["baz"].(map[string]interface{})["quux"].([]interface{})[0],
		data["baz"].(map[string]interface{})["xyzzy"],
		data["baz"].(map[string]interface{})["ipsum"],
	}

	for i, path := range paths {
		expected := expecteds[i]

		actual := v.Get(path...)

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("%v != %v\n", actual, expected)
		}
	}
}

func TestGetComment(t *testing.T) {
	expecteds := []string{
		"Root comment",
		"This is foo",
		"This is baz",
		"This is quux",
		"This is quux[0]",
		"This is xyzzy",
		"",
	}

	for i, path := range paths {
		expected := expecteds[i]

		actual := v.GetComment(path...)

		if actual != expected {
			t.Errorf("%v != %v\n", actual, expected)
		}
	}
}
