package cfn

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/aws-cloudformation/rain/cfn/spec"
	"github.com/aws-cloudformation/rain/cfn/spec/models"
	"github.com/aws-cloudformation/rain/cfn/value"
)

// Error holds errors messages pertaining to a cfn.Template
type Error struct {
	template *Template
	comments map[interface{}]interface{}
}

// Check validates a cfn.Template against the cloudformation spec
func (t Template) Check() (value.Interface, bool) {
	out := value.New(t.Map())

	rs := out.Get("Resources")
	if rs == nil {
		out.SetComment("Template has no Resources")
		return out, false
	}

	resources, ok := rs.(*value.Map)
	if !ok {
		rs.SetComment("Not a map!")
		return out, false
	}

	return out, checkResources(resources)
}

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

func checkResources(resources *value.Map) bool {
	outOk := true

	// Check resources
	for _, name := range resources.Keys() {
		resource, ok := resources.Get(name).(*value.Map)
		if !ok {
			resources.Get(name).SetComment("Resource must be a map")
			outOk = false
			continue
		}

		t := resource.Get("Type")
		if t == nil {
			resource.SetComment("Resource must define a Type")
			outOk = false
			continue
		}

		typeName, ok := t.Value().(string)
		if !ok {
			t.SetComment(fmt.Sprintf("Type must be a string"))
			outOk = false
			continue
		}

		if typeName == "AWS::CloudFormation::CustomResource" || strings.HasPrefix(typeName, "Custom::") {
			continue
		}

		rSpec, ok := spec.Cfn.ResourceTypes[typeName]
		if !ok {
			t.SetComment(fmt.Sprintf("Unknown type '%s'", typeName))
			continue // Just a warning
		}

		// Create empty properties if there aren't any
		if resource.Get("Properties") == nil {
			resource.Set("Properties", make(map[string]interface{}))
		}

		props, ok := resource.Get("Properties").(*value.Map)
		if !ok {
			resource.Get("Properties").SetComment("Properties must be a map")
			outOk = false
			continue
		}

		outOk = checkProperties(rSpec, props) && outOk
	}

	return outOk
}

func checkProperties(rSpec *models.ResourceType, props *value.Map) bool {
	outOk := true

	// Check for missing required properties
	missing := make([]string, 0)
	for name, prop := range rSpec.Properties {
		if prop.Required && props.Get(name) == nil {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		props.SetComment(fmt.Sprintf("Missing required properties: %s", strings.Join(missing, ", ")))
		outOk = false
	}

	// Check present properties for validity
	for _, name := range props.Keys() {
		prop := props.Get(name)

		pSpec, ok := rSpec.Properties[name]
		if !ok {
			prop.SetComment(fmt.Sprintf("Unknown property '%s'", name))
			outOk = false
		} else {
			outOk = checkProperty(pSpec, prop) && outOk
		}
	}

	return outOk
}

func checkProperty(pSpec *models.Property, prop value.Interface) bool {
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
	return true
}

func checkMap(pSpec *models.Property, prop value.Interface) bool {
	return true // TODO
}

func checkList(pSpec *models.Property, prop value.Interface) bool {
	return true // TODO
}

func checkString(pSpec *models.Property, prop value.Interface) bool {
	if isIntrinsic(prop.Value()) {
		return true
	}
	_, ok := prop.Value().(string)
	if !ok {
		prop.SetComment("Should be a string")
		return false
	}
	return true
}

func checkBool(pSpec *models.Property, prop value.Interface) bool {
	if isIntrinsic(prop.Value()) {
		return true
	}
	_, ok := prop.Value().(bool)
	if !ok {
		prop.SetComment("Should be a boolean")
		return false
	}
	return true
}

func checkJSON(pSpec *models.Property, prop value.Interface) bool {
	_, ok := prop.(*value.Map)
	if ok {
		return true
	}

	s, ok := prop.Value().(string)
	if !ok {
		prop.SetComment("Should be a JSON string")
		return false
	}

	var out interface{}
	err := json.Unmarshal([]byte(s), &out)
	if err != nil {
		prop.SetComment("Invalid JSON")
		return false
	}

	return true
}

func checkNumber(pSpec *models.Property, prop value.Interface) bool {
	if isIntrinsic(prop.Value()) {
		return true
	}

	switch prop.Value().(type) {
	case float64, int:
		return true
	default:
		prop.SetComment("Should be a number")
		return false
	}
}

func checkTimestamp(pSpec *models.Property, prop value.Interface) bool {
	_, ok := prop.Value().(string)
	if !ok {
		prop.SetComment("Should be a timestamp")
		return false
	}
	return true
}

/*
type PropertyType struct {
type ResourceType struct {
type Property struct {
type Attribute struct {
*/
