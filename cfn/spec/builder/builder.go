package builder

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cfn/spec/models"
)

const (
	policyDocument           = "PolicyDocument"
	assumeRolePolicyDocument = "AssumeRolePolicyDocument"
	optionalTag              = "Optional"
	changeMeTag              = "CHANGEME"
)

// Builder generates a template from its Spec
type Builder struct {
	Spec                      models.Spec
	IncludeOptionalProperties bool
	BuildIamPolicies          bool
}

var iamBuilder IamBuilder

var emptyProp = models.Property{}

func init() {
	iamBuilder = NewIamBuilder()
}

func (b Builder) newResource(resourceType string) (map[string]interface{}, map[string]interface{}) {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("error building resource type '%s': %w", resourceType, r))
		}
	}()

	rSpec, ok := b.Spec.ResourceTypes[resourceType]
	if !ok {
		panic(fmt.Errorf("no such resource type '%s'", resourceType))
	}

	// Generate properties
	properties := make(map[string]interface{})
	comments := make(map[string]interface{})
	for name, pSpec := range rSpec.Properties {
		if b.IncludeOptionalProperties || pSpec.Required {
			if b.BuildIamPolicies && (name == policyDocument || name == assumeRolePolicyDocument) {
				properties[name], comments[name] = iamBuilder.Policy()
			} else {
				properties[name], comments[name] = b.newProperty(resourceType, name, pSpec)
			}
		}
	}

	return map[string]interface{}{
			"Type":       resourceType,
			"Properties": properties,
		}, map[string]interface{}{
			"Properties": comments,
		}
}

func (b Builder) newProperty(resourceType, propertyName string, pSpec *models.Property) (interface{}, interface{}) {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("error building property %s.%s: %w", resourceType, propertyName, r))
		}
	}()

	// Correct badly-formed entries
	if pSpec.PrimitiveType == models.TypeMap {
		pSpec.PrimitiveType = models.TypeEmpty
		pSpec.Type = models.TypeMap
	}

	// Primitive types
	if pSpec.PrimitiveType != models.TypeEmpty {
		if pSpec.Required {
			return b.newPrimitive(pSpec.PrimitiveType), nil
		}

		return b.newPrimitive(pSpec.PrimitiveType), optionalTag
	}

	if pSpec.Type == models.TypeList || pSpec.Type == models.TypeMap {
		var value interface{}
		var subComments interface{}

		// Calculate a single item example
		if pSpec.PrimitiveItemType != models.TypeEmpty {
			value = b.newPrimitive(pSpec.PrimitiveItemType)
		} else if pSpec.ItemType != models.TypeEmpty {
			value, subComments = b.newPropertyType(resourceType, pSpec.ItemType)
		} else {
			value = changeMeTag
		}

		if pSpec.Type == models.TypeList {
			// Returning a list

			var comments []interface{}
			if subComments != nil {
				comments = []interface{}{subComments}
			}

			return []interface{}{value}, comments
		}

		// Returning a map
		comments := make(map[string]interface{})
		if subComments != nil {
			comments[changeMeTag] = subComments
		}

		return map[string]interface{}{changeMeTag: value}, comments
	}

	// Fall through to property types
	output, comments := b.newPropertyType(resourceType, pSpec.Type)

	if !pSpec.Required {
		if comments == nil {
			comments = optionalTag
		} else if commentMap, ok := comments.(map[string]interface{}); ok {
			commentMap[""] = optionalTag
		}
	}

	return output, comments
}

func (b Builder) newPrimitive(primitiveType string) interface{} {
	switch primitiveType {
	case "String":
		return changeMeTag
	case "Integer":
		return 0
	case "Double":
		return 0.0
	case "Long":
		return 0.0
	case "Boolean":
		return false
	case "Timestamp":
		return "1970-01-01 00:00:00"
	case "Json":
		return "{\"JSON\": \"CHANGEME\"}"
	default:
		panic(fmt.Errorf("unimplemented primitive type '%s'", primitiveType))
	}
}

func (b Builder) newPropertyType(resourceType, propertyType string) (interface{}, interface{}) {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("error building property type '%s.%s': %w", resourceType, propertyType, r))
		}
	}()

	var ptSpec *models.PropertyType
	var ok bool

	// If we've used a property from another resource type
	// switch to that resource type for now
	parts := strings.Split(propertyType, ".")
	if len(parts) == 2 {
		resourceType = parts[0]
	}

	ptSpec, ok = b.Spec.PropertyTypes[propertyType]
	if !ok {
		ptSpec, ok = b.Spec.PropertyTypes[resourceType+"."+propertyType]
	}
	if !ok {
		panic(fmt.Errorf("unimplemented property type '%s.%s'", resourceType, propertyType))
	}

	// Deal with the case that a property type is directly a plain property
	// for example AWS::Glue::SecurityConfiguration.S3Encryptions
	if ptSpec.Property != emptyProp {
		return b.newProperty(resourceType, propertyType, &ptSpec.Property)
	}

	comments := make(map[string]interface{})

	// Generate properties
	properties := make(map[string]interface{})
	for name, pSpec := range ptSpec.Properties {
		if !pSpec.Required {
			comments[name] = optionalTag
		}

		if b.BuildIamPolicies && (name == policyDocument || name == assumeRolePolicyDocument) {
			properties[name], comments[name] = iamBuilder.Policy()
		} else if pSpec.Type == propertyType || pSpec.ItemType == propertyType {
			properties[name] = make(map[string]interface{})
		} else {
			properties[name], _ = b.newProperty(resourceType, name, pSpec)
		}
	}

	return properties, comments
}
