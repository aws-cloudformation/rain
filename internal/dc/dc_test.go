package dc

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
	"gopkg.in/yaml.v3"

	"github.com/google/go-cmp/cmp"
)

type deployConfigTestCase struct {
	name           string
	envVars        map[string]string
	envTags        map[string]string
	cliTags        []string
	cliParams      []string
	configContent  string
	expectedParams map[string]string
	expectedTags   map[string]string
}

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

func TestGetDeployConfigWithEnvironmentVariables(t *testing.T) {
	templateYaml := `
Parameters:
  InstanceType:
    Type: String
    Description: EC2 instance type
    Default: t3.micro
  Environment:
    Type: String
    Default: dev
  VpcId:
    Type: String
    Default: vpc-default
`

	tests := []deployConfigTestCase{
		{
			name: "only environment variables",
			envVars: map[string]string{
				"RAIN_VAR_InstanceType": "t3.medium",
				"RAIN_VAR_Environment":  "production",
			},
			cliTags:   []string{},
			cliParams: []string{},
			expectedParams: map[string]string{
				"InstanceType": "t3.medium",
				"Environment":  "production",
				"VpcId":        "vpc-default",
			},
		},
		{
			name: "env vars with CLI override",
			envVars: map[string]string{
				"RAIN_VAR_InstanceType": "t3.medium",
				"RAIN_VAR_Environment":  "production",
			},
			cliTags:   []string{},
			cliParams: []string{"InstanceType=t3.large"},
			expectedParams: map[string]string{
				"InstanceType": "t3.large",
				"Environment":  "production",
				"VpcId":        "vpc-default",
			},
		},
		{
			name: "env vars with config file override",
			envVars: map[string]string{
				"RAIN_VAR_InstanceType": "t3.medium",
				"RAIN_VAR_Environment":  "production",
				"RAIN_VAR_VpcId":        "vpc-env",
			},
			cliTags:       []string{},
			cliParams:     []string{},
			configContent: "Parameters:\n  InstanceType: t3.xlarge\n  Environment: staging\n",
			expectedParams: map[string]string{
				"InstanceType": "t3.xlarge",
				"Environment":  "staging",
				"VpcId":        "vpc-env",
			},
		},
		{
			name: "all three sources - CLI wins",
			envVars: map[string]string{
				"RAIN_VAR_InstanceType": "t3.medium",
				"RAIN_VAR_Environment":  "production",
				"RAIN_VAR_VpcId":        "vpc-env",
			},
			cliTags:       []string{},
			cliParams:     []string{"InstanceType=t3.2xlarge"},
			configContent: "Parameters:\n  Environment: staging\n  VpcId: vpc-config\n",
			expectedParams: map[string]string{
				"InstanceType": "t3.2xlarge",
				"Environment":  "staging",
				"VpcId":        "vpc-config",
			},
		},
		{
			name:      "no env vars",
			envVars:   map[string]string{},
			cliTags:   []string{},
			cliParams: []string{"InstanceType=t3.small"},
			expectedParams: map[string]string{
				"InstanceType": "t3.small",
				"Environment":  "dev",
				"VpcId":        "vpc-default",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			for k, v := range tt.envTags {
				t.Setenv(k, v)
			}

			var node yaml.Node
			if err := yaml.Unmarshal([]byte(templateYaml), &node); err != nil {
				t.Fatalf("Failed to parse template: %v", err)
			}

			template := &cft.Template{
				Node: &node,
			}

			stack := types.Stack{}

			configFilePath := ""
			if tt.configContent != "" {
				tempFile, err := os.CreateTemp(t.TempDir(), "test-config-*.yaml")
				if err != nil {
					t.Fatalf("Failed to create temp config file: %v", err)
				}
				defer tempFile.Close()

				if _, err := tempFile.WriteString(tt.configContent); err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}
				configFilePath = tempFile.Name()
			}

			dc, err := GetDeployConfig(
				tt.cliTags,
				tt.cliParams,
				configFilePath,
				"test.yaml",
				template,
				stack,
				false,
				true,
				false,
			)
			if err != nil {
				t.Fatalf("GetDeployConfig failed: %v", err)
			}

			if tt.expectedParams != nil {
				actualParams := make(map[string]string)
				for _, param := range dc.Params {
					if param.ParameterValue != nil {
						actualParams[*param.ParameterKey] = *param.ParameterValue
					}
				}

				for key, expectedValue := range tt.expectedParams {
					if actualValue, ok := actualParams[key]; !ok {
						t.Errorf("Expected parameter %s not found", key)
					} else if actualValue != expectedValue {
						t.Errorf("Parameter %s: expected %s, got %s", key, expectedValue, actualValue)
					}
				}

				for key := range actualParams {
					if _, ok := tt.expectedParams[key]; !ok {
						t.Errorf("Unexpected parameter %s with value %s", key, actualParams[key])
					}
				}
			}

			if tt.expectedTags != nil {
				for key, expectedValue := range tt.expectedTags {
					if actualValue, ok := dc.Tags[key]; !ok {
						t.Errorf("Expected tag %s not found", key)
					} else if actualValue != expectedValue {
						t.Errorf("Tag %s: expected %s, got %s", key, expectedValue, actualValue)
					}
				}

				for key := range dc.Tags {
					if _, ok := tt.expectedTags[key]; !ok {
						t.Errorf("Unexpected tag %s with value %s", key, dc.Tags[key])
					}
				}
			}
		})
	}
}

func TestGetDeployConfigWithEnvironmentTags(t *testing.T) {
	templateYaml := `
Parameters:
  InstanceType:
    Type: String
    Default: t3.micro
`

	tests := []deployConfigTestCase{
		{
			name: "only environment tags",
			envTags: map[string]string{
				"RAIN_DEFAULT_TAG_ManagedBy":  "Rain",
				"RAIN_DEFAULT_TAG_CostCenter": "engineering",
			},
			cliTags:   []string{},
			cliParams: []string{},
			expectedTags: map[string]string{
				"ManagedBy":  "Rain",
				"CostCenter": "engineering",
			},
		},
		{
			name: "env tags with CLI override",
			envTags: map[string]string{
				"RAIN_DEFAULT_TAG_ManagedBy":  "Rain",
				"RAIN_DEFAULT_TAG_CostCenter": "engineering",
			},
			cliTags:   []string{"ManagedBy=CLI"},
			cliParams: []string{},
			expectedTags: map[string]string{
				"ManagedBy":  "CLI",
				"CostCenter": "engineering",
			},
		},
		{
			name: "env tags with config file override",
			envTags: map[string]string{
				"RAIN_DEFAULT_TAG_ManagedBy":  "Rain",
				"RAIN_DEFAULT_TAG_CostCenter": "engineering",
				"RAIN_DEFAULT_TAG_Owner":      "env-owner",
			},
			cliTags:       []string{},
			cliParams:     []string{},
			configContent: "Tags:\n  ManagedBy: ConfigFile\n  CostCenter: finance\n",
			expectedTags: map[string]string{
				"ManagedBy":  "ConfigFile",
				"CostCenter": "finance",
				"Owner":      "env-owner",
			},
		},
		{
			name: "all three sources - CLI wins",
			envTags: map[string]string{
				"RAIN_DEFAULT_TAG_ManagedBy":  "Rain",
				"RAIN_DEFAULT_TAG_CostCenter": "engineering",
				"RAIN_DEFAULT_TAG_Owner":      "env-owner",
			},
			cliTags:       []string{"ManagedBy=CLI"},
			cliParams:     []string{},
			configContent: "Tags:\n  CostCenter: finance\n  Owner: config-owner\n",
			expectedTags: map[string]string{
				"ManagedBy":  "CLI",
				"CostCenter": "finance",
				"Owner":      "config-owner",
			},
		},
		{
			name:      "no env tags",
			envTags:   map[string]string{},
			cliTags:   []string{"ManagedBy=CLI"},
			cliParams: []string{},
			expectedTags: map[string]string{
				"ManagedBy": "CLI",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			for k, v := range tt.envTags {
				t.Setenv(k, v)
			}

			var node yaml.Node
			if err := yaml.Unmarshal([]byte(templateYaml), &node); err != nil {
				t.Fatalf("Failed to parse template: %v", err)
			}

			template := &cft.Template{
				Node: &node,
			}

			stack := types.Stack{}

			configFilePath := ""
			if tt.configContent != "" {
				tempFile, err := os.CreateTemp(t.TempDir(), "test-config-*.yaml")
				if err != nil {
					t.Fatalf("Failed to create temp config file: %v", err)
				}
				defer tempFile.Close()

				if _, err := tempFile.WriteString(tt.configContent); err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}
				configFilePath = tempFile.Name()
			}

			dc, err := GetDeployConfig(
				tt.cliTags,
				tt.cliParams,
				configFilePath,
				"test.yaml",
				template,
				stack,
				false,
				true,
				false,
			)
			if err != nil {
				t.Fatalf("GetDeployConfig failed: %v", err)
			}

			if tt.expectedParams != nil {
				actualParams := make(map[string]string)
				for _, param := range dc.Params {
					if param.ParameterValue != nil {
						actualParams[*param.ParameterKey] = *param.ParameterValue
					}
				}

				for key, expectedValue := range tt.expectedParams {
					if actualValue, ok := actualParams[key]; !ok {
						t.Errorf("Expected parameter %s not found", key)
					} else if actualValue != expectedValue {
						t.Errorf("Parameter %s: expected %s, got %s", key, expectedValue, actualValue)
					}
				}

				for key := range actualParams {
					if _, ok := tt.expectedParams[key]; !ok {
						t.Errorf("Unexpected parameter %s with value %s", key, actualParams[key])
					}
				}
			}

			if tt.expectedTags != nil {
				for key, expectedValue := range tt.expectedTags {
					if actualValue, ok := dc.Tags[key]; !ok {
						t.Errorf("Expected tag %s not found", key)
					} else if actualValue != expectedValue {
						t.Errorf("Tag %s: expected %s, got %s", key, expectedValue, actualValue)
					}
				}

				for key := range dc.Tags {
					if _, ok := tt.expectedTags[key]; !ok {
						t.Errorf("Unexpected tag %s with value %s", key, dc.Tags[key])
					}
				}
			}
		})
	}
}
