package format

import "sort"

var orders = map[string][]string{
	"Template": {
		"AWSTemplateFormatVersion",
		"Description",
		"Metadata",
		"Parameters",
		"Mappings",
		"Conditions",
		"Transform",
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

	var seen bool
	for _, key := range keys {
		seen = false

		for _, o := range orders[name] {
			if key == o {
				known = append(known, key)
				seen = true
				break
			}
		}

		if !seen {
			unknown = append(unknown, key)
		}
	}

	return append(known, unknown...)
}

func (p *encoder) sortKeys() []string {
	keys := sortMapKeys(p.currentValue.(map[string]interface{}))

	switch {
	case len(p.path) == 0:
		return sortAs(keys, "Template")
	case len(p.path) == 0:
		return sortAs(keys, "Parameter")
	case p.path[0] == "Transform" || p.path[len(p.path)-1] == "Fn::Transform":
		return sortAs(keys, "Transform")
	case p.path[0] == "Resources" && len(p.path) == 2:
		return sortAs(keys, "Resource")
	case p.path[0] == "Outputs" && len(p.path) == 2:
		return sortAs(keys, "Outputs")
	case len(p.path) > 2 && p.path[len(p.path)-2] == "Policies":
		return sortAs(keys, "Policy")
	case p.path[len(p.path)-1] == "PolicyDocument" || p.path[len(p.path)-1] == "AssumeRolePolicyDocument":
		return sortAs(keys, "PolicyDocument")
	case len(p.path) > 2 && p.path[len(p.path)-2] == "Statement":
		return sortAs(keys, "PolicyStatement")
	case len(p.path) > 3 && p.path[0] == "Resources" && p.path[2] == "Properties":
		return sortAs(keys, "ResourceProperties")
	default:
		return keys
	}
}
