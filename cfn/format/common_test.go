package format

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestIntrinsicKeys(t *testing.T) {
	cases := []map[string]interface{}{
		{"foo": "bar"},
		{"Ref": "cake"},
		{"Fn::IncludeIf": "banana"},
		{"Fn::IncludeIf": "banana", "Ref": "cake"},
		{"Fn::Sub": "The cake is a lie"},
		{"Fn::GetAtt": []string{"foo", "bar"}},
		{"Fn::NotARealFn": "But we'll take it anyway"},
		{"Func::Join": "We're not taking this one"},
	}

	expecteds := []string{
		"",
		"Ref",
		"Fn::IncludeIf",
		"",
		"Fn::Sub",
		"Fn::GetAtt",
		"Fn::NotARealFn",
		"",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual, _ := intrinsicKey(testCase)

		if actual != expected {
			t.Errorf("%q != %q\n", actual, expected)
		}
	}
}

func TestFormatString(t *testing.T) {
	cases := []string{
		"foo",
		"\"quoted\"",
		" space at start",
		"space at end ",
		" space and\nnewline",
		"multi\nline",
		"\nstarts with a newline",
		"ends with a single newline\n",
		"ends with\nmultiple newlines\n\n\n",
		"\n", // All whitespace
	}

	expecteds := []string{
		"foo",
		"\"\\\"quoted\\\"\"",
		"\" space at start\"",
		"\"space at end \"",
		"\" space and\\nnewline\"",
		"|-\n  multi\n  line",
		"|-\n  \n  starts with a newline",
		"|\n  ends with a single newline\n  ",
		"|+\n  ends with\n  multiple newlines\n  \n  ",
		"\"\\n\"",
	}

	// Check they're formatted as expected
	for i, testCase := range cases {
		expected := expecteds[i]

		actual := formatString(testCase)

		if actual != expected {
			t.Errorf("%q != %q\n", actual, expected)
		}
	}

	// And check yaml parses them back the same way
	for i, testCase := range expecteds {
		expected := cases[i]

		var actual map[string]interface{}
		err := yaml.Unmarshal([]byte(fmt.Sprintf("foo: %s\n", testCase)), &actual)
		if err != nil {
			t.Error(err)
		}

		if actual["foo"] != expected {
			t.Errorf("%q != %q\n", actual, expected)
		}
	}
}
