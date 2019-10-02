package cfn

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/aws-cloudformation/rain/cfn/spec"
	"github.com/aws-cloudformation/rain/cfn/spec/models"
)

type CfnError struct {
	template *Template
	comments map[interface{}]interface{}
}

func (t Template) Check() error {
	rs, ok := t.Map()["Resources"]
	if !ok {
		return errors.New("Template has no resources")
	}

	resources, ok := rs.(map[string]interface{})
	if !ok {
		return errors.New("Resources isn't a map!")
	}

	// Check resources
	for name, r := range resources {
		resource, ok := r.(map[string]interface{})
		if !ok {
			return fmt.Errorf("Resource '%s' isn't a map!", name)
		}

		t, ok := resource["Type"]
		if !ok {
			return fmt.Errorf("Resource '%s' has no type!", name)
		}

		typeName, ok := t.(string)
		if !ok {
			return fmt.Errorf("Resource '%s' has an invalid type: %s", name, t)
		}

		p, ok := resource["Properties"]
		if ok {
			props, ok := p.(map[string]interface{})
			if !ok {
				return fmt.Errorf("Resource '%s' has invalid properties: %s", name, p)
			}

			rType, ok := spec.Cfn.ResourceTypes[typeName]
			if !ok {
				return fmt.Errorf("Unknown resource type: %s", typeName)
			}

			err := checkResourceType(rType, props)
			if err != nil {
				return fmt.Errorf("Resource '%s' has errors: %s", name, err)
			}
		}
	}

	return nil
}

func checkResourceType(r models.ResourceType, in map[string]interface{}) error {
	for name, p := range in {
		prop, ok := r.Properties[name]
		if !ok {
			return fmt.Errorf("Unknown property name: %s", name)
		}

		err := checkProperty(prop, p)
		if err != nil {
			return fmt.Errorf("Parameter '%s' has errors: %s", name, err)
		}
	}

	return nil
}

func checkProperty(p models.Property, in interface{}) error {
	switch p.Type {
	case "Map":
		return checkMap(p, in)
	case "List":
		return checkList(p, in)
	}

	switch p.PrimitiveType {
	case "String":
		return checkString(p, in)
	case "Boolean":
		return checkBool(p, in)
	case "Json":
		return checkJson(p, in)
	case "Double", "Long", "Integer":
		return checkNumber(p, in)
	case "Timestamp":
		return checkTimestamp(p, in)
	case "Map":
		return checkMap(p, in)
	case "List":
		return checkList(p, in)
	}

	// TODO: Property types
	return nil
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

func checkMap(p models.Property, in interface{}) error {
	return nil
}

func checkList(p models.Property, in interface{}) error {
	return nil
}

func checkString(p models.Property, in interface{}) error {
	if isIntrinsic(in) {
		return nil
	}
	_, ok := in.(string)
	if !ok {
		return fmt.Errorf("Not a string: %s", in)
	}
	return nil
}

func checkBool(p models.Property, in interface{}) error {
	if isIntrinsic(in) {
		return nil
	}
	_, ok := in.(bool)
	if !ok {
		return fmt.Errorf("Not a boolean: %s", in)
	}
	return nil
}

func checkJson(p models.Property, in interface{}) error {
	_, ok := in.(map[string]interface{})
	if ok {
		return nil
	}

	s, ok := in.(string)
	if !ok {
		return fmt.Errorf("Not a JSON string: %s", in)
	}

	var out interface{}
	err := json.Unmarshal([]byte(s), &out)
	if err != nil {
		return fmt.Errorf("Not a JSON string: %s", in)
	}

	return nil
}

func checkNumber(p models.Property, in interface{}) error {
	if isIntrinsic(in) {
		return nil
	}
	_, ok := in.(float64)
	if !ok {
		return fmt.Errorf("Not a number: %s", in)
	}
	return nil
}

func checkTimestamp(p models.Property, in interface{}) error {
	_, ok := in.(string)
	if !ok {
		return fmt.Errorf("Not a timestamp: %s", in)
	}
	return nil
}

/*
type PropertyType struct {
type ResourceType struct {
type Property struct {
type Attribute struct {
*/
