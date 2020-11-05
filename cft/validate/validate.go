// Package validate contains functionality for checking a cft.Template against
// the published CloudFormation specification - as represented in the cft.spec package
package validate

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/spec"
)

func isIntrinsic(in interface{}) bool {
	m, ok := in.(map[string]interface{})
	if !ok {
		return false
	}

	if len(m) != 1 {
		return false
	}

	keys := reflect.ValueOf(m).MapKeys()
	if keys[0].String() == "Ref" || strings.HasPrefix(keys[0].String(), "Fn::") {
		return true
	}

	return false
}

// Template validates a cft.Template against the cloudformation spec.
// Any problems are returned as a slice of pointers to cft.Comment
// which can be passed directly to cft.Template.AddComments to
// annotate a template with its validation errors if desired.
func Template(t cft.Template) []*cft.Comment {
	in := t.Map()

	rs, ok := in["Resources"]
	if !ok {
		errs := make(errors, 0)
		errs.add("Template has no resources")
		return []*cft.Comment(errs)
	}

	resources, ok := rs.(map[string]interface{})
	if !ok {
		errs := make(errors, 0)
		errs.add("Resources must be a map", "Resources")
		return []*cft.Comment(errs)
	}

	errs := checkResources(resources)
	for _, err := range errs {
		if err.Path == nil {
			err.Path = make([]interface{}, 0)
		}

		err.Path = append([]interface{}{"Resources"}, err.Path...)
	}
	return []*cft.Comment(errs)
}

func checkResources(resources map[string]interface{}) errors {
	errs := make(errors, 0)

	// Sort resource names
	resourceNames := make([]string, 0)
	for name := range resources {
		resourceNames = append(resourceNames, name)
	}
	sort.Strings(resourceNames)

	// Check resources
	for _, name := range resourceNames {
		r := resources[name]

		resource, ok := r.(map[string]interface{})
		if !ok {
			errs.add("Resource must be a map", name)
			continue
		}

		t, ok := resource["Type"]
		if !ok {
			errs.add("Resource must have a Type", name)
			continue
		}

		typeName, ok := t.(string)
		if !ok {
			errs.add("Type must be a string", name, "Type")
			continue
		}

		if typeName == "AWS::CloudFormation::CustomResource" || strings.HasPrefix(typeName, "Custom::") {
			continue
		}

		rSpec, ok := spec.Cfn.ResourceTypes[typeName]
		if !ok {
			errs.add(fmt.Sprintf("Unknown type '%s'", typeName), name, "Type")
		}

		// Create empty properties if there aren't any
		if _, ok := resource["Properties"]; !ok {
			resource["Properties"] = make(map[string]interface{})
		}

		props, ok := resource["Properties"].(map[string]interface{})
		if !ok {
			errs.add("Properties must be a map", name, "Properties")
			continue
		}

		if rSpec != nil {
			for _, err := range checkProperties(rSpec, props) {
				errs.add(err.Value, append([]interface{}{name, "Properties"}, err.Path...)...)
			}
		}
	}

	return errs
}

func checkProperties(rSpec *spec.ResourceType, props map[string]interface{}) errors {
	errs := make(errors, 0)

	// Check for missing required properties
	missing := make([]string, 0)
	for name, prop := range rSpec.Properties {
		if prop.Required && props[name] == nil {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		errs.add(fmt.Sprintf("Missing required properties: %s", strings.Join(missing, ", ")))
	}

	// Sort property names
	propNames := make([]string, 0)
	for name := range props {
		propNames = append(propNames, name)
	}
	sort.Strings(propNames)

	// Check present properties for validity
	for _, name := range propNames {
		prop := props[name]

		pSpec, ok := rSpec.Properties[name]
		if !ok {
			errs.add(fmt.Sprintf("Unknown property '%s'", name), name)
		} else {
			for _, err := range checkProperty(pSpec, prop) {
				errs.add(err.Value, append([]interface{}{name}, err.Path...))
			}
		}
	}

	return errs
}

func checkProperty(pSpec *spec.Property, prop interface{}) errors {
	switch pSpec.Type {
	case "Map":
		return checkMap(pSpec, prop)
	case "List":
		return checkList(pSpec, prop)
	}

	switch pSpec.PrimitiveType {
	case "String":
		return checkString(pSpec, prop)
	case "Boolean":
		return checkBool(pSpec, prop)
	case "Json":
		return checkJSON(pSpec, prop)
	case "Double", "Long", "Integer":
		return checkNumber(pSpec, prop)
	case "Timestamp":
		return checkTimestamp(pSpec, prop)
	case "Map":
		return checkMap(pSpec, prop)
	case "List":
		return checkList(pSpec, prop)
	}

	// TODO: Property types
	return make(errors, 0)
}

func checkMap(pSpec *spec.Property, prop interface{}) errors {
	return make(errors, 0) // TODO
}

func checkList(pSpec *spec.Property, prop interface{}) errors {
	return make(errors, 0) // TODO
}

func checkString(pSpec *spec.Property, prop interface{}) errors {
	errs := make(errors, 0)

	if isIntrinsic(prop) {
		return errs
	}

	_, ok := prop.(string)
	if !ok {
		errs.add("Should be a string")
	}

	return errs
}

func checkBool(pSpec *spec.Property, prop interface{}) errors {
	errs := make(errors, 0)

	if isIntrinsic(prop) {
		return errs
	}

	_, ok := prop.(bool)
	if !ok {
		errs.add("Should be a boolean")
	}

	return errs
}

func checkJSON(pSpec *spec.Property, prop interface{}) errors {
	errs := make(errors, 0)

	_, ok := prop.(map[string]interface{})
	if ok {
		return errs
	}

	s, ok := prop.(string)
	if ok {
		var out interface{}
		err := json.Unmarshal([]byte(s), &out)
		if err != nil {
			errs.add("Invalid JSON")
		}
	}

	return errs
}

func checkNumber(pSpec *spec.Property, prop interface{}) errors {
	errs := make(errors, 0)

	if isIntrinsic(prop) {
		return errs
	}

	switch prop.(type) {
	case float64, int:
		return errs
	default:
		errs.add("Should be a number")
		return errs
	}
}

func checkTimestamp(pSpec *spec.Property, prop interface{}) errors {
	errs := make(errors, 0)
	_, ok := prop.(string)
	if !ok {
		errs.add("Should be a timestamp")
	}
	return errs
}

/*
type PropertyType struct {
type ResourceType struct {
type Property struct {
type Attribute struct {
*/
