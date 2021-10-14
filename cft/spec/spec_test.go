package spec_test

import (
	"encoding/json"
	"fmt"
	"github.com/aws-cloudformation/rain/cft/spec"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"testing"
)

// stripPatterns removes custom regexes
// as the cfn-spec uses extended regexes
// which the jsonschema package does not support
func stripPatterns(in interface{}) {
	switch v := in.(type) {
	case map[string]interface{}:
		delete(v, "patternProperties")
		for key, value := range v {
			if key == "pattern" || key == "format" {
				v[key] = ".*"
			} else {
				stripPatterns(value)
			}
		}
	case []interface{}:
		for _, child := range v {
			stripPatterns(child)
		}
	}
}

// TestCfn confirms that all cfn schemas are valid jsonschema
func TestCfn(t *testing.T) {
	for typeName, schema := range spec.Cfn {
		stripPatterns(schema)

		data, err := json.Marshal(schema)
		if err != nil {
			t.Error(err)
		}

		_, err = jsonschema.CompileString("schema.json", string(data))
		if err != nil {
			t.Error(fmt.Errorf("%s: %w", typeName, err))
		}
	}
}
