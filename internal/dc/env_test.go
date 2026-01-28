package dc

import (
	"testing"
)

func TestGetParametersFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected map[string]string
	}{
		{
			name: "single parameter",
			envVars: map[string]string{
				"RAIN_VAR_InstanceType": "t3.medium",
			},
			expected: map[string]string{
				"InstanceType": "t3.medium",
			},
		},
		{
			name: "multiple parameters",
			envVars: map[string]string{
				"RAIN_VAR_InstanceType": "t3.medium",
				"RAIN_VAR_Environment":  "production",
				"RAIN_VAR_VpcId":        "vpc-12345",
			},
			expected: map[string]string{
				"InstanceType": "t3.medium",
				"Environment":  "production",
				"VpcId":        "vpc-12345",
			},
		},
		{
			name: "empty value",
			envVars: map[string]string{
				"RAIN_VAR_EmptyParam": "",
			},
			expected: map[string]string{
				"EmptyParam": "",
			},
		},
		{
			name: "value with equals signs",
			envVars: map[string]string{
				"RAIN_VAR_ConnectionString": "host=localhost;user=admin;password=secret=123",
			},
			expected: map[string]string{
				"ConnectionString": "host=localhost;user=admin;password=secret=123",
			},
		},
		{
			name: "value with commas",
			envVars: map[string]string{
				"RAIN_VAR_SubnetIds": "subnet-1,subnet-2,subnet-3",
			},
			expected: map[string]string{
				"SubnetIds": "subnet-1,subnet-2,subnet-3",
			},
		},
		{
			name: "value with special characters",
			envVars: map[string]string{
				"RAIN_VAR_SpecialChars": "foo@bar.com:8080/path?query=value&other=123",
			},
			expected: map[string]string{
				"SpecialChars": "foo@bar.com:8080/path?query=value&other=123",
			},
		},
		{
			name: "parameter name with underscores",
			envVars: map[string]string{
				"RAIN_VAR_DB_Instance_Class": "db.t3.medium",
			},
			expected: map[string]string{
				"DB_Instance_Class": "db.t3.medium",
			},
		},
		{
			name: "mixed with non-RAIN_VAR variables",
			envVars: map[string]string{
				"RAIN_VAR_Param1": "value1",
				"OTHER_VAR":       "ignored",
				"PATH":            "/usr/bin",
				"RAIN_VAR_Param2": "value2",
			},
			expected: map[string]string{
				"Param1": "value1",
				"Param2": "value2",
			},
		},
		{
			name: "case sensitivity - wrong case ignored",
			envVars: map[string]string{
				"RAIN_VAR_Param":  "correct",
				"rain_var_param":  "ignored",
				"Rain_Var_Param":  "ignored",
				"RAIN_var_Param2": "ignored",
			},
			expected: map[string]string{
				"Param": "correct",
			},
		},
		{
			name: "empty name after prefix (edge case)",
			envVars: map[string]string{
				"RAIN_VAR_": "should_be_ignored",
			},
			expected: map[string]string{},
		},
		{
			name:     "no RAIN_VAR variables",
			envVars:  map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "value with newlines",
			envVars: map[string]string{
				"RAIN_VAR_MultiLine": "line1\nline2\nline3",
			},
			expected: map[string]string{
				"MultiLine": "line1\nline2\nline3",
			},
		},
		{
			name: "value with quotes",
			envVars: map[string]string{
				"RAIN_VAR_Quoted": `"quoted value"`,
			},
			expected: map[string]string{
				"Quoted": `"quoted value"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			result := GetParametersFromEnv()
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d variables, got %d", len(tt.expected), len(result))
			}

			for k, expectedValue := range tt.expected {
				if actualValue, ok := result[k]; !ok {
					t.Errorf("Expected key %q not found in result", k)
				} else if actualValue != expectedValue {
					t.Errorf("For key %q: expected value %q, got %q", k, expectedValue, actualValue)
				}
			}

			for k := range result {
				if _, ok := tt.expected[k]; !ok {
					t.Errorf("Unexpected key %q found in result with value %q", k, result[k])
				}
			}
		})
	}
}

func TestGetDefaultTagsFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected map[string]string
	}{
		{
			name: "single tag",
			envVars: map[string]string{
				"RAIN_DEFAULT_TAG_ManagedBy": "Rain",
			},
			expected: map[string]string{
				"ManagedBy": "Rain",
			},
		},
		{
			name: "multiple tags",
			envVars: map[string]string{
				"RAIN_DEFAULT_TAG_ManagedBy":  "Rain",
				"RAIN_DEFAULT_TAG_CostCenter": "engineering",
				"RAIN_DEFAULT_TAG_Owner":      "platform-team",
			},
			expected: map[string]string{
				"ManagedBy":  "Rain",
				"CostCenter": "engineering",
				"Owner":      "platform-team",
			},
		},
		{
			name: "empty tag value",
			envVars: map[string]string{
				"RAIN_DEFAULT_TAG_EmptyTag": "",
			},
			expected: map[string]string{
				"EmptyTag": "",
			},
		},
		{
			name: "tag with special characters in value",
			envVars: map[string]string{
				"RAIN_DEFAULT_TAG_Email": "team@example.com",
			},
			expected: map[string]string{
				"Email": "team@example.com",
			},
		},
		{
			name: "tag name with underscores",
			envVars: map[string]string{
				"RAIN_DEFAULT_TAG_Cost_Center": "engineering",
			},
			expected: map[string]string{
				"Cost_Center": "engineering",
			},
		},
		{
			name: "mixed with non-RAIN_DEFAULT_TAG variables",
			envVars: map[string]string{
				"RAIN_DEFAULT_TAG_Tag1": "value1",
				"OTHER_TAG":             "ignored",
				"RAIN_VAR_Param":        "ignored",
				"RAIN_DEFAULT_TAG_Tag2": "value2",
			},
			expected: map[string]string{
				"Tag1": "value1",
				"Tag2": "value2",
			},
		},
		{
			name: "case sensitivity",
			envVars: map[string]string{
				"RAIN_DEFAULT_TAG_Tag":  "correct",
				"rain_default_tag_tag":  "ignored",
				"Rain_Default_Tag_Tag2": "ignored",
			},
			expected: map[string]string{
				"Tag": "correct",
			},
		},
		{
			name: "empty name after prefix",
			envVars: map[string]string{
				"RAIN_DEFAULT_TAG_": "ignored",
			},
			expected: map[string]string{},
		},
		{
			name:     "no default tags",
			envVars:  map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "AWS-style tag keys",
			envVars: map[string]string{
				"RAIN_DEFAULT_TAG_aws:cloudformation:stack-name": "my-stack",
			},
			expected: map[string]string{
				"aws:cloudformation:stack-name": "my-stack",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			result := GetDefaultTagsFromEnv()
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tags, got %d", len(tt.expected), len(result))
			}

			for k, expectedValue := range tt.expected {
				if actualValue, ok := result[k]; !ok {
					t.Errorf("Expected key %q not found in result", k)
				} else if actualValue != expectedValue {
					t.Errorf("For key %q: expected value %q, got %q", k, expectedValue, actualValue)
				}
			}

			for k := range result {
				if _, ok := tt.expected[k]; !ok {
					t.Errorf("Unexpected key %q found in result with value %q", k, result[k])
				}
			}
		})
	}
}
