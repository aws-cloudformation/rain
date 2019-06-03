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
	}

	expecteds := []string{
		"",
		"Ref",
		"Fn::IncludeIf",
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
	}

	expecteds := []string{
		"foo",
		"\"\\\"quoted\\\"\"",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual := formatString(testCase)

		if actual != expected {
			t.Errorf("%q != %q\n", actual, expected)
		}
	}
}
