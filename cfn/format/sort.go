package format

import (
	"sort"

	"github.com/aws-cloudformation/rain/cfn"
)

var orders = map[string][]string{
	"Template": {
		"AWSTemplateFormatVersion",
		"Description",
		"Transform",
		"Parameters",
		"Metadata",
		"Mappings",
		"Conditions",
		"Resources",
		"Outputs",
	},
	"Parameter": {
		"Type",
		"Default",
	},
	"Transform": {
		"Name",
		"Parameters",
	},
	"Resource": {
		"Type",
	},
	"Outputs": {
		"Description",
		"Value",
		"Export",
	},
	"Policy": {
		"PolicyName",
		"PolicyDocument",
	},
	"PolicyDocument": {
		"Version",
		"Id",
		"Statement",
	},
	"PolicyStatement": {
		"Sid",
		"Effect",
		"Principal",
		"NotPrincipal",
		"Action",
		"NotAction",
		"Resource",
		"NotResource",
		"Condition",
	},
	"ResourceProperties": {
		"Name",
		"Description",
		"Type",
	},
	"Swagger": {
		"swagger",
		"info",
	},
}

func sortMapKeys(value map[string]interface{}) []string {
	keys := make([]string, 0)
	for key, _ := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}

func sortAs(keys []string, name string) []string {
	// Map keys to known and unknown list
	known := make([]string, 0)
	unknown := make([]string, 0)

	seen := make(map[string]bool)

	for _, o := range orders[name] {
		for _, key := range keys {
			if key == o {
				known = append(known, key)
				seen[key] = true
				break
			}
		}
	}

	for _, key := range keys {
		if !seen[key] {
			unknown = append(unknown, key)
		}
	}

	return append(known, unknown...)
}

func (p *encoder) sortKeys() []string {
	var keys []string

	if p.currentValue == nil {
		return keys
	}

	keys = sortMapKeys(p.currentValue.Value().(map[string]interface{}))

	// Specific length paths
	if len(p.path) == 0 {
		return sortAs(keys, "Template")
	}

	// Resources
	if len(p.path) == 1 {
		if p.path[0] == "Resources" {
			if t, ok := p.value.Value().(map[string]interface{}); ok {
				g := cfn.Template(t).Graph()

				output := make([]string, 0)
				for _, item := range g.Nodes() {
					el := item.(cfn.Element)

					if el.Type == "Resources" {
						output = append(output, el.Name)
					}
				}

				return output
			}
		}
	}

	// Top-level elements
	if len(p.path) == 2 {
		switch p.path[0] {
		case "Parameters":
			return sortAs(keys, "Parameter")
		case "Resources":
			return sortAs(keys, "Resource")
		case "Outputs":
			return sortAs(keys, "Outputs")
		}
	}

	// Known array types
	if len(p.path) > 2 {
		switch p.path[len(p.path)-2] {
		case "Policies":
			return sortAs(keys, "Policy")
		case "Statement":
			return sortAs(keys, "PolicyStatement")
		}
	}

	// Paths that can live anywhere
	if p.path[0] == "Transform" || p.path[len(p.path)-1] == "Fn::Transform" {
		return sortAs(keys, "Transform")
	}
	if p.path[len(p.path)-1] == "PolicyDocument" || p.path[len(p.path)-1] == "AssumeRolePolicyDocument" {
		return sortAs(keys, "PolicyDocument")
	}
	if p.path[len(p.path)-1] == "DefinitionBody" {
		return sortAs(keys, "Swagger")
	}

	// General resource properties
	if len(p.path) > 3 {
		if p.path[0] == "Resources" && p.path[2] == "Properties" {
			return sortAs(keys, "ResourceProperties")
		}
	}

	return keys
}
