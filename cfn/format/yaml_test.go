package format_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/aws-cloudformation/rain/cfn/value"
)

func TestYamlDirectValues(t *testing.T) {
	cases := []interface{}{
		1,
		"foo",
		[]interface{}{1, "foo"},
		map[string]interface{}{"foo": "bar", "baz": "quux"},
		float64(500000000),
		float64(500000000.98765),
	}

	expecteds := []string{
		"1",
		"foo",
		"- 1\n- foo",
		"baz: quux\n\nfoo: bar",
		"500000000",
		"500000000.98765",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual := format.Anything(testCase, format.Options{})

		if actual != expected {
			t.Errorf("from %T %v:\n%#v != %#v\n", testCase, testCase, actual, expected)
		}
	}
}

func TestCompactYaml(t *testing.T) {
	cases := []interface{}{
		map[string]interface{}{"foo": "bar", "baz": "quux"},
	}

	expecteds := []string{
		"baz: quux\nfoo: bar",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual := format.Anything(testCase, format.Options{Compact: true})

		if actual != expected {
			t.Errorf("from %T %v:\n%#v != %#v\n", testCase, testCase, actual, expected)
		}
	}
}

func TestYamlScalars(t *testing.T) {
	cases := []map[string]interface{}{
		{"foo": 1},
		{"foo": 1.0},
		{"foo": 1.234},
		{"foo": float64(500000000)},
		{"foo": "32"},
		{"foo": "032"},
		{"foo": "32.0"},
		{"foo": "500000000"},
		{"foo": "hello"},
		{"foo": true},
		{"foo": false},
	}

	expecteds := []string{
		"foo: 1",
		"foo: 1",
		"foo: 1.234",
		"foo: 500000000",
		"foo: \"32\"",
		"foo: \"032\"",
		"foo: \"32.0\"",
		"foo: \"500000000\"",
		"foo: hello",
		"foo: true",
		"foo: false",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual := format.Anything(testCase, format.Options{})

		if actual != expected {
			t.Errorf("from %T %v:\n%#v != %#v\n", testCase, testCase, actual, expected)
		}
	}
}

func TestYamlList(t *testing.T) {
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
		"foo: []",
		"foo:\n  - 1",
		"foo:\n  - 1\n  - foo\n  - true",
		"foo:\n  - - foo\n    - bar\n  - baz",
		"foo:\n  - - - foo\n      - bar\n    - baz\n  - quux",
		"foo:\n  - foo: bar\n  - baz: quux\n    mooz: xyzzy",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual := format.Anything(testCase, format.Options{})

		if actual != expected {
			t.Errorf("from %T %v:\n%#v != %#v\n", testCase, testCase, actual, expected)
		}
	}
}

func TestYamlMap(t *testing.T) {
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
		{
			"32": map[string]interface{}{
				"64": "Numeric key",
			},
		},
	}

	expecteds := []string{
		"{}",
		"foo: bar",
		"baz: quux\n\nfoo: bar",
		"foo:\n  bar: baz\n\nquux: mooz",
		"alpha: beta\n\nfoo:\n  bar:\n    baz: quux\n\n  mooz: xyzzy",
		"foo:\n  - bar\n  - baz\n\nquux:\n  - mooz",
		"\"32\":\n  \"64\": Numeric key",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual := format.Anything(testCase, format.Options{})

		if actual != expected {
			t.Errorf("from %T %v:\n%#v != %#v\n", testCase, testCase, actual, expected)
		}
	}
}

func TestCfnYaml(t *testing.T) {
	cases := []map[string]interface{}{
		{
			"Quux":       "mooz",
			"Parameters": "baz",
			"Foo":        "bar",
			"Resources":  "xyzzy",
		},
	}

	expecteds := []string{
		"Parameters: baz\n\nResources: xyzzy\n\nFoo: bar\n\nQuux: mooz",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual := format.Anything(testCase, format.Options{})

		if actual != expected {
			t.Errorf("from %T %v:\n%#v != %#v\n", testCase, testCase, actual, expected)
		}
	}
}

func TestIntrinsics(t *testing.T) {
	cases := []map[string]interface{}{
		{
			"foo": map[string]interface{}{
				"Ref": "bar",
			},
		},
		{
			"foo": map[string]interface{}{
				"Fn::Sub": []interface{}{
					"The ${key} is a ${value}",
					map[string]interface{}{
						"key":   "cake",
						"value": "lie",
					},
				},
			},
		},
		{
			"foo": map[string]interface{}{
				"Fn::GetAtt": []interface{}{
					"Cake",
					"Lie",
				},
			},
		},
	}

	expecteds := []string{
		"foo: !Ref bar",
		"foo: !Sub\n  - The ${key} is a ${value}\n  - key: cake\n    value: lie",
		"foo: !GetAtt Cake.Lie",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual := format.Anything(testCase, format.Options{})

		if actual != expected {
			t.Errorf("from %T %v:\n%#v != %#v\n", testCase, testCase, actual, expected)
		}
	}
}

func TestStrings(t *testing.T) {
	cases := []map[string]interface{}{
		{"foo": "foo"},
		{"foo": "*"},
		{"foo": "* bar"},
		{"foo": "2012-05-02"},
		{"foo": "today is 2012-05-02"},
		{"foo": ": thing"},
		{"foo": "Yes"},
		{"foo": "No"},
		{"foo": "multi\nline"},
	}

	expecteds := []string{
		"foo: foo",
		"foo: \"*\"",
		"foo: \"* bar\"",
		"foo: \"2012-05-02\"",
		"foo: today is 2012-05-02",
		"foo: \": thing\"",
		"foo: \"Yes\"",
		"foo: \"No\"",
		"foo: |\n  multi\n  line",
	}

	for i, testCase := range cases {
		expected := expecteds[i]

		actual := format.Anything(testCase, format.Options{})

		if actual != expected {
			t.Errorf("from %T %v:\n%#v != %#v\n", testCase, testCase, actual, expected)
		}
	}
}

func TestYamlComments(t *testing.T) {
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
		"# Top-level comment\nbaz:\n  quux: mooz\n\nfoo: bar\n\nxyzzy:\n  - lorem",
		"baz:\n  quux: mooz\n\nfoo: bar  # This is foo\n\nxyzzy:\n  - lorem",
		"baz:  # This is baz\n  quux: mooz\n\nfoo: bar\n\nxyzzy:\n  - lorem",
		"baz:\n  quux: mooz  # This is quux\n\nfoo: bar\n\nxyzzy:\n  - lorem",
		"baz:\n  quux: mooz\n\nfoo: bar\n\nxyzzy:  # This is xyzzy\n  - lorem",
		"baz:\n  quux: mooz\n\nfoo: bar\n\nxyzzy:\n  - lorem  # This is lorem",
	}

	for i, comment := range comments {
		expected := expecteds[i]

		v := value.New(data)
		v.Get(comment.path...).SetComment(comment.value)

		actual := format.Value(v, format.Options{})

		if actual != expected {
			t.Errorf("from %q != %q\n", actual, expected)
		}
	}
}
