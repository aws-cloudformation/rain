package value_test

import (
	"reflect"
	"testing"

	"github.com/aws-cloudformation/rain/cfn/value"
)

var original = map[string]interface{}{
	"foo": "bar",
	"baz": []interface{}{42, "answer"},
	"quux": map[string]interface{}{
		"mooz": "xyzzy",
	},
}

func TestGet(t *testing.T) {
	v := value.New(original)

	for _, testCase := range []struct {
		path     []interface{}
		expected interface{}
	}{
		{[]interface{}{}, original},
		{[]interface{}{"foo"}, original["foo"]},
		{[]interface{}{"baz"}, original["baz"]},
		{[]interface{}{"baz", 0}, original["baz"].([]interface{})[0]},
		{[]interface{}{"baz", 1}, original["baz"].([]interface{})[1]},
		{[]interface{}{"quux"}, original["quux"]},
		{[]interface{}{"quux", "mooz"}, original["quux"].(map[string]interface{})["mooz"]},
	} {
		actual := v.Get(testCase.path...).Value()

		if !reflect.DeepEqual(actual, testCase.expected) {
			t.Errorf("'%#v' is not '%#v'", actual, testCase.expected)
		}
	}
}

func TestSetComments(t *testing.T) {
	v := value.New(original)

	comments := []struct {
		path  []interface{}
		value string
	}{
		{[]interface{}{}, "Top-level"},
		{[]interface{}{"foo"}, "Comment on foo"},
		{[]interface{}{"baz"}, "Comment on baz"},
		{[]interface{}{"baz", 0}, "Life, the universe, and everything"},
		{[]interface{}{"baz", 1}, "Ultimate"},
		{[]interface{}{"quux"}, "Quuxy comment"},
		{[]interface{}{"quux", "mooz"}, "Secret word"},
	}

	for _, comment := range comments {
		v.Get(comment.path...).SetComment(comment.value)

		if v.Get(comment.path...).Comment() != comment.value {
			t.Fail()
		}
	}
}
