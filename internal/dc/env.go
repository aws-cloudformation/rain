package dc

import (
	"os"
	"strings"
)

const (
	// EnvVarPrefix is the prefix used for environment variables that define template variables.
	// Variables with this prefix are automatically loaded and made available during template processing.
	EnvVarPrefix = "RAIN_VAR_"

	// DefaultTagPrefix is the prefix used for environment variables that define default tags.
	// Variables with this prefix are automatically loaded and made available during template processing.
	DefaultTagPrefix = "RAIN_DEFAULT_TAG_"
)

// GetParametersFromEnv scans the environment for RAIN_VAR_ prefixed environment
// variables and returns them as a parameter map. The prefix is removed from the keys.
//
// Example:
//
//	export RAIN_VAR_InstanceType="t3.medium"
//	export RAIN_VAR_Environment="production"
//
// Returns:
//
//	map[string]string{"InstanceType": "t3.medium", "Environment": "production"}
func GetParametersFromEnv() map[string]string {
	return getEnvironmentVariablesWithPrefix(EnvVarPrefix)
}

// GetDefaultTagsFromEnv scans the environment for RAIN_DEFAULT_TAG_ prefixed environment
// variables and returns them as a tag map. The prefix is removed from the keys.
//
// Example:
//
//	export RAIN_DEFAULT_TAG_ManagedBy="Rain"
//	export RAIN_DEFAULT_TAG_CostCenter="engineering"
//
// Returns:
//
//	map[string]string{"ManagedBy": "Rain", "CostCenter": "engineering"}
func GetDefaultTagsFromEnv() map[string]string {
	return getEnvironmentVariablesWithPrefix(DefaultTagPrefix)
}

func getEnvironmentVariablesWithPrefix(prefix string) map[string]string {
	result := make(map[string]string)

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, prefix) {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimPrefix(parts[0], prefix)
				if key != "" {
					result[key] = parts[1]
				}
			}
		}
	}

	return result
}
