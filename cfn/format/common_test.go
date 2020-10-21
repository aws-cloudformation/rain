package format

import (
	"testing"
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
		"ends with a newline\n",
		"ends with\nmultiple newlines\n\n\n",
	}

	expecteds := []string{
		"foo",
		"\"\\\"quoted\\\"\"",
		"\" space at start\"",
		"\"space at end \"",
		"\" space and\\nnewline\"",
		"|-\n  multi\n  line",
		"|-\n\n  starts with a newline",
		"|\n  ends with a newline\n",
		"|+\n  ends with\n  multiple newlines\n\n\n",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual := formatString(testCase)

		if actual != expected {
			t.Errorf("%q != %q\n", actual, expected)
		}
	}
}
