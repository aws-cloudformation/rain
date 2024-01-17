package dc

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestListToMap(t *testing.T) {
	testCases := []struct {
		input    []string
		expected map[string]string
	}{
		{[]string{"Foo=bar"}, map[string]string{"Foo": "bar"}},
		{[]string{"Foo=bar", "Baz=quux"}, map[string]string{"Foo": "bar", "Baz": "quux"}},
		{[]string{"Foo=bar", "baz"}, map[string]string{"Foo": "bar,baz"}},
		{[]string{"Foo=bar", "Baz=quux", "mooz"}, map[string]string{"Foo": "bar", "Baz": "quux,mooz"}},
		{[]string{"Foo=bar", "Baz=quux", "mooz", "Xyzzy=garply"}, map[string]string{"Foo": "bar", "Baz": "quux,mooz", "Xyzzy": "garply"}},
		{[]string{"Foo=bar", "Baz=quux", "Mooz=xyzzy", "garply"}, map[string]string{"Foo": "bar", "Baz": "quux", "Mooz": "xyzzy,garply"}},
	}

	for _, testCase := range testCases {
		actual := ListToMap("test", testCase.input)

		if d := cmp.Diff(testCase.expected, actual); d != "" {
			t.Errorf(d)
		}
	}
}

func TestConfigFromStack(t *testing.T) {
	// Prepare test cases with a stack name and a map of parameters and tags
	testCases := []struct {
		testCaseName     string            // The name of the stack
		params           map[string]string // The expected parameters
		tags             map[string]string // The expected tags
		yamlString       string            // The expected yaml string
		exactStringMatch bool              // If true, the config must match the yamlString exactly. Otherwise, the config is validated tag by tag, and param by param
	}{
		{"test-stack-one-tag-one-param", map[string]string{"Foo": "bar"}, map[string]string{"Baz": "quux"}, "Parameters:\n  Foo: bar\nTags:\n  Baz: quux\n", true},
		{"test-stack-multiple-tags-and-params-1", map[string]string{"Foo": "bar", "Baz": "quux"}, map[string]string{"Mooz": "xyzzy", "Garply": "thud"}, "Parameters:\n  Foo: bar\n  Baz: quux\nTags:\n  Mooz: xyzzy\n  Garply: thud\n", false},
		{"test-stack-multiple-tags-and-params-2", map[string]string{"Foo": "bar", "Baz": "quux", "Mooz": "xyzzy"}, map[string]string{"Garply": "thud"}, "Parameters:\n  Foo: bar\n  Baz: quux\n  Mooz: xyzzy\nTags:\n  Garply: thud\n", false},
		{"test-stack-multiple-tags-and-params-3", map[string]string{"Foo": "bar", "Baz": "quux", "Mooz": "xyzzy", "Garply": "thud"}, map[string]string{}, "Parameters:\n  Foo: bar\n  Baz: quux\n  Mooz: xyzzy\n  Garply: thud\nTags: {}\n", false},
		{"test-stack-with-no-tags", map[string]string{"Foo": "bar"}, map[string]string{}, "Parameters:\n  Foo: bar\nTags: {}\n", false},
		{"test-stack-with-no-params", map[string]string{}, map[string]string{"Foo": "bar", "Baz": "quux", "Mooz": "xyzzy", "Garply": "thud"}, "Parameters: {}\nTags:\n  Foo: bar\n  Baz: quux\n  Mooz: xyzzy\n  Garply: thud\n", false},
		{"test-stack-with-no-tags-and-params", map[string]string{}, map[string]string{}, "Parameters: {}\nTags: {}\n", true},
	}
	// Iterate over test cases
	for _, testCase := range testCases {
		// Create a stack with the given name, parameters and tags
		stack := types.Stack{
			Tags:       []types.Tag{},
			Parameters: []types.Parameter{},
		}
		for key, value := range testCase.tags {
			stack.Tags = append(stack.Tags, types.Tag{Key: ptr.String(key), Value: ptr.String(value)})
		}
		for key, value := range testCase.params {
			stack.Parameters = append(stack.Parameters, types.Parameter{ParameterKey: ptr.String(key), ParameterValue: ptr.String(value)})
		}

		// Get the config from the stack
		config, err := ConfigFromStack(stack)
		if err != nil {
			t.Errorf("case %s - expected no error, got '%s'", testCase.testCaseName, err)
		}

		// Check if the config matches the expected string
		if testCase.exactStringMatch {
			if config != testCase.yamlString {
				t.Errorf("case %s - expected '%s', got '%s'", testCase.testCaseName, testCase.yamlString, config)
			}
		} else {
			// For each tag in testcase, check if it is present in the config
			for key, value := range testCase.tags {
				expectedString := fmt.Sprintf("%s: %s", key, value)
				if strings.Contains(config, fmt.Sprintf("%s: %s", key, value)) == false {
					t.Errorf("case %s - expected '%s' in '%s'", testCase.testCaseName, expectedString, testCase.yamlString)
				}
			}

			// For each param in testcase, check if it is present in the config
			for key, value := range testCase.params {
				expectedString := fmt.Sprintf("%s: %s", key, value)
				if strings.Contains(config, expectedString) == false {
					t.Errorf("case %s - expected '%s' in '%s'", testCase.testCaseName, expectedString, testCase.yamlString)
				}
			}
		}

	}
}
