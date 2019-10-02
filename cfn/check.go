package cfn

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/aws-cloudformation/rain/cfn/spec"
	"github.com/aws-cloudformation/rain/cfn/spec/models"
	"github.com/aws-cloudformation/rain/cfn/value"
)

type CfnError struct {
	template *Template
	comments map[interface{}]interface{}
}

func (t Template) Check() (value.Interface, bool) {
	out := value.New(t.Map())

	rs := out.Get("Resources")
	if rs == nil {
		out.SetComment("Missing Resources!")
		return out, false
	}

	resources, ok := rs.(*value.Map)
	if !ok {
		rs.SetComment("Not a map!")
		return out, false
	}

	return out, checkResources(resources)
}

func checkResources(resources *value.Map) bool {
	outOk := true

	// Check resources
	for _, name := range resources.Keys() {
		resource := resources.Get(name)
		_, ok := resources.Get(name).(*value.Map)
		if !ok {
			resource.SetComment("Not a map!")
			outOk = false
			continue
		}

		t := resource.Get("Type")
		if t == nil {
			resource.SetComment("Missing Type!")
			outOk = false
			continue
		}

		typeName, ok := t.Value().(string)
		if !ok {
			t.SetComment("Invalid type!")
			outOk = false
			continue
		}

		rSpec, ok := spec.Cfn.ResourceTypes[typeName]
		if !ok {
			t.SetComment("Unknown type")
			outOk = false
			continue
		}

		p := resource.Get("Properties")
		if p != nil {
			props, ok := p.(*value.Map)
			if !ok {
				p.SetComment("Not a map!")
				outOk = false
				continue
			}

			outOk = outOk && checkProperties(rSpec, props)
		}
	}

	return outOk
}

func checkProperties(rSpec models.ResourceType, props *value.Map) bool {
	outOk := true

	for _, name := range props.Keys() {
		pSpec, ok := rSpec.Properties[name]
		if !ok {
			props.Get(name).SetComment("Unknown property")
			outOk = false
		}

		outOk = outOk && checkProperty(pSpec, props.Get(name))
	}

	return outOk
}

func checkProperty(pSpec models.Property, prop value.Interface) bool {
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
		return checkJson(pSpec, prop)
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

func checkMap(pSpec models.Property, prop value.Interface) bool {
	return true // TODO
}

func checkList(pSpec models.Property, prop value.Interface) bool {
	return true // TODO
}

func checkString(pSpec models.Property, prop value.Interface) bool {
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

func checkBool(pSpec models.Property, prop value.Interface) bool {
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

func checkJson(pSpec models.Property, prop value.Interface) bool {
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

func checkNumber(pSpec models.Property, prop value.Interface) bool {
	if isIntrinsic(prop.Value()) {
		return true
	}
	_, ok := prop.Value().(float64)
	if !ok {
		prop.SetComment("Should be a number")
		return false
	}
	return true
}

func checkTimestamp(pSpec models.Property, prop value.Interface) bool {
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
