package value_test

import (
	"fmt"
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

func TestWalk(t *testing.T) {
	v := value.New(original)

	expected := []value.Node{
		{[]interface{}{}, v.Get()},
		{[]interface{}{"baz"}, v.Get("baz")},
		{[]interface{}{"baz", 0}, v.Get("baz", 0)},
		{[]interface{}{"baz", 1}, v.Get("baz", 1)},
		{[]interface{}{"foo"}, v.Get("foo")},
		{[]interface{}{"quux"}, v.Get("quux")},
		{[]interface{}{"quux", "mooz"}, v.Get("quux", "mooz")},
	}

	actual := v.Nodes()

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%#v", actual)
	}
}

func TestNodeString(t *testing.T) {
	v := value.New(original)

	expected := []string{
		"[]: {...}",
		"[baz]: [...]",
		"[baz/0]: 42",
		"[baz/1]: answer",
		"[foo]: bar",
		"[quux]: {...}",
		"[quux/mooz]: xyzzy",
	}

	actual := make([]string, 0)
	for _, node := range v.Nodes() {
		actual = append(actual, node.String())
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%#v", actual)
	}
}

func TestCommentedNodeString(t *testing.T) {
	v := value.New(original)

	expected := []string{
		"[]: {...}  # Comment0",
		"[baz]: [...]  # Comment1",
		"[baz/0]: 42  # Comment2",
		"[baz/1]: answer  # Comment3",
		"[foo]: bar  # Comment4",
		"[quux]: {...}  # Comment5",
		"[quux/mooz]: xyzzy  # Comment6",
	}

	actual := make([]string, 0)
	for i, node := range v.Nodes() {
		node.Content.SetComment(fmt.Sprint("Comment", i))
		actual = append(actual, node.String())
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%#v", actual)
	}
}
