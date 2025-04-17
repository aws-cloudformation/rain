package cft

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// unmarshalYaml is a helper function that unmarshals a YAML string and returns
// the mapping node
func unmarshalYaml(t *testing.T, yamlStr string) *yaml.Node {
	node := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlStr), node)
	if err != nil {
		t.Fatalf("Failed to unmarshal test YAML: %v", err)
	}
	return node.Content[0]
}

func TestModuleConfigProperties(t *testing.T) {
	// Create a test ModuleConfig with a PropertiesNode
	propertiesYaml := `
foo: bar
baz: 
  qux: quux
numbers:
  - 1
  - 2
  - 3
`
	config := ModuleConfig{
		Name:           "TestModule",
		PropertiesNode: unmarshalYaml(t, propertiesYaml),
	}

	// Test the Properties method
	props := config.Properties()

	// Check that the properties were decoded correctly
	if props["foo"] != "bar" {
		t.Errorf("Expected foo=bar, got %v", props["foo"])
	}

	// Check nested property
	bazMap, ok := props["baz"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected baz to be a map, got %T", props["baz"])
	} else if bazMap["qux"] != "quux" {
		t.Errorf("Expected baz.qux=quux, got %v", bazMap["qux"])
	}

	// Check array property
	numbers, ok := props["numbers"].([]interface{})
	if !ok {
		t.Errorf("Expected numbers to be an array, got %T", props["numbers"])
	} else if len(numbers) != 3 {
		t.Errorf("Expected numbers to have 3 elements, got %d", len(numbers))
	}
}

func TestModuleConfigOverrides(t *testing.T) {
	// Create a test ModuleConfig with an OverridesNode
	overridesYaml := `
Resource1:
  Properties:
    Name: overridden
Resource2:
  Metadata:
    Comment: added
`
	config := ModuleConfig{
		Name:          "TestModule",
		OverridesNode: unmarshalYaml(t, overridesYaml),
	}

	// Test the Overrides method
	overrides := config.Overrides()

	// Check that the overrides were decoded correctly
	resource1, ok := overrides["Resource1"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected Resource1 to be a map, got %T",
			overrides["Resource1"])
	} else {
		props, ok := resource1["Properties"].(map[string]interface{})
		if !ok {
			t.Errorf("Expected Resource1.Properties to be a map, got %T",
				resource1["Properties"])
		} else if props["Name"] != "overridden" {
			t.Errorf("Expected Resource1.Properties.Name=overridden, got %v",
				props["Name"])
		}
	}
}

func TestResourceOverridesNode(t *testing.T) {
	// Create a test ModuleConfig with an OverridesNode
	overridesYaml := `
Resource1:
  Properties:
    Name: overridden
Resource2:
  Metadata:
    Comment: added
`
	config := ModuleConfig{
		Name:          "TestModule",
		OverridesNode: unmarshalYaml(t, overridesYaml),
	}

	// Test ResourceOverridesNode for an existing resource
	resource1Node := config.ResourceOverridesNode("Resource1")
	if resource1Node == nil {
		t.Errorf("Expected to get a node for Resource1, got nil")
	}

	// Test ResourceOverridesNode for a non-existent resource
	nonExistentNode := config.ResourceOverridesNode("NonExistentResource")
	if nonExistentNode != nil {
		t.Errorf("Expected to get nil for NonExistentResource, got %v",
			nonExistentNode)
	}

	// Test ResourceOverridesNode when OverridesNode is nil
	configWithNilOverrides := ModuleConfig{
		Name:          "TestModule",
		OverridesNode: nil,
	}
	nilResult := configWithNilOverrides.ResourceOverridesNode("Resource1")
	if nilResult != nil {
		t.Errorf("Expected nil when OverridesNode is nil, got %v", nilResult)
	}
}

func TestParseModuleConfig(t *testing.T) {
	// Create a test Template
	template := &Template{}

	// Test case 1: Basic module configuration
	basicModuleYaml := `
Source: ./module.yaml
Properties:
  Name: test
Overrides:
  Resource1:
    Properties:
      Name: overridden
`
	basicConfig, err := template.ParseModuleConfig("BasicModule",
		unmarshalYaml(t, basicModuleYaml))
	if err != nil {
		t.Errorf("ParseModuleConfig failed: %v", err)
	}

	if basicConfig.Name != "BasicModule" {
		t.Errorf("Expected Name=BasicModule, got %s", basicConfig.Name)
	}
	if basicConfig.Source != "./module.yaml" {
		t.Errorf("Expected Source=./module.yaml, got %s", basicConfig.Source)
	}
	if basicConfig.PropertiesNode == nil {
		t.Errorf("Expected PropertiesNode to be non-nil")
	}
	if basicConfig.OverridesNode == nil {
		t.Errorf("Expected OverridesNode to be non-nil")
	}

	// Test case 2: Module with Map
	mapModuleYaml := `
Source: ./module.yaml
ForEach: !Ref MyList
Properties:
  Name: test-${MapValue}
`
	mapConfig, err := template.ParseModuleConfig("MapModule",
		unmarshalYaml(t, mapModuleYaml))
	if err != nil {
		t.Errorf("ParseModuleConfig failed: %v", err)
	}

	if mapConfig.Name != "MapModule" {
		t.Errorf("Expected Name=MapModule, got %s", mapConfig.Name)
	}
	if mapConfig.Map == nil {
		t.Errorf("Expected Map to be non-nil")
	}

	// Test case 3: Invalid node type
	invalidNodeYaml := `- not a mapping node`
	_, err = template.ParseModuleConfig("InvalidModule",
		unmarshalYaml(t, invalidNodeYaml))
	if err == nil {
		t.Errorf("Expected error for invalid node type, got nil")
	}
}

func TestParseModuleConfigWithFnForEach(t *testing.T) {
	// Create a test Template
	template := &Template{}

	// Test case: Fn::ForEach module configuration
	forEachYaml := `
- item
- !Ref MyList
- OutputKey:
    Source: ./module.yaml
    Properties:
      Name: !Sub test-${item}
`
	forEachConfig, err := template.ParseModuleConfig("Fn::ForEach:LoopModule",
		unmarshalYaml(t, forEachYaml))
	if err != nil {
		t.Errorf("ParseModuleConfig failed for Fn::ForEach: %v", err)
	}

	if forEachConfig.FnForEach == nil {
		t.Errorf("Expected FnForEach to be non-nil")
	}

	if forEachConfig.FnForEach.LoopName != "LoopModule" {
		t.Errorf("Expected LoopName=LoopModule, got %s",
			forEachConfig.FnForEach.LoopName)
	}

	if forEachConfig.FnForEach.Identifier != "item" {
		t.Errorf("Expected Identifier=item, got %s",
			forEachConfig.FnForEach.Identifier)
	}

	if forEachConfig.FnForEach.OutputKey != "OutputKey" {
		t.Errorf("Expected OutputKey=OutputKey, got %s",
			forEachConfig.FnForEach.OutputKey)
	}

	// Test invalid Fn::ForEach (wrong number of elements)
	invalidForEachYaml := `
- item
- !Ref MyList
`
	_, err = template.ParseModuleConfig("Fn::ForEach:InvalidLoop",
		unmarshalYaml(t, invalidForEachYaml))
	if err == nil {
		t.Errorf("Expected error for invalid Fn::ForEach, got nil")
	}

	// Test invalid Fn::ForEach (invalid output mapping)
	invalidOutputYaml := `
- item
- !Ref MyList
- InvalidOutput
`
	_, err = template.ParseModuleConfig("Fn::ForEach:InvalidOutput",
		unmarshalYaml(t, invalidOutputYaml))
	if err == nil {
		t.Errorf("Expected error for invalid output mapping, got nil")
	}
}

func TestFnForEachOutputKeyHasIdentifier(t *testing.T) {
	// Test cases for OutputKeyHasIdentifier
	testCases := []struct {
		name       string
		identifier string
		outputKey  string
		expected   bool
	}{
		{
			name:       "Dollar syntax present",
			identifier: "item",
			outputKey:  "test-${item}",
			expected:   true,
		},
		{
			name:       "Ampersand syntax present",
			identifier: "item",
			outputKey:  "test-&{item}",
			expected:   true,
		},
		{
			name:       "No identifier",
			identifier: "item",
			outputKey:  "test-static",
			expected:   false,
		},
		{
			name:       "Different identifier",
			identifier: "item",
			outputKey:  "${different}",
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fnForEach := FnForEach{
				Identifier: tc.identifier,
				OutputKey:  tc.outputKey,
			}

			result := fnForEach.OutputKeyHasIdentifier()
			if result != tc.expected {
				t.Errorf("Expected OutputKeyHasIdentifier()=%v, got %v",
					tc.expected, result)
			}
		})
	}
}

func TestFnForEachReplaceIdentifier(t *testing.T) {
	// Test cases for ReplaceIdentifier
	testCases := []struct {
		name       string
		identifier string
		input      string
		key        string
		expected   string
	}{
		{
			name:       "Replace dollar syntax",
			identifier: "item",
			input:      "test-${item}-suffix",
			key:        "value1",
			expected:   "test-value1-suffix",
		},
		{
			name:       "Replace ampersand syntax",
			identifier: "item",
			input:      "test-&{item}-suffix",
			key:        "value1",
			expected:   "test-value1-suffix",
		},
		{
			name:       "Replace multiple occurrences",
			identifier: "item",
			input:      "${item}-middle-${item}",
			key:        "value1",
			expected:   "value1-middle-value1",
		},
		{
			name:       "No replacement needed",
			identifier: "item",
			input:      "static-string",
			key:        "value1",
			expected:   "static-string",
		},
		{
			name:       "Mixed syntax",
			identifier: "item",
			input:      "${item}-and-&{item}",
			key:        "value1",
			expected:   "value1-and-value1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			result := ReplaceIdentifier(tc.input, tc.key, tc.identifier)
			if result != tc.expected {
				t.Errorf("Expected ReplaceIdentifier()=%q, got %q",
					tc.expected, result)
			}
		})
	}
}
