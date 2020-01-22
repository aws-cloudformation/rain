package builder

import (
	"github.com/aws-cloudformation/rain/cfn/spec/models"
)

const (
	PolicyDocument           = "PolicyDocument"
	AssumeRolePolicyDocument = "AssumeRolePolicyDocument"
	OptionalTag              = "Optional"
	ChangeMeTag              = "CHANGEME"
)

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
	rSpec, ok := b.Spec.ResourceTypes[resourceType]
	if !ok {
		panic("No such resource type: " + resourceType)
	}

	// Generate properties
	properties := make(map[string]interface{})
	comments := make(map[string]interface{})
	for name, pSpec := range rSpec.Properties {
		if b.IncludeOptionalProperties || pSpec.Required {
			if b.BuildIamPolicies && (name == PolicyDocument || name == AssumeRolePolicyDocument) {
				properties[name], comments[name] = iamBuilder.Policy()
			} else {
				properties[name], comments[name] = b.newProperty(resourceType, pSpec)
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

func (b Builder) newProperty(resourceType string, pSpec models.Property) (interface{}, interface{}) {
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

		return b.newPrimitive(pSpec.PrimitiveType), OptionalTag
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
			value = ChangeMeTag
		}

		if pSpec.Type == models.TypeList {
			// Returning a list

			var comments []interface{}
			if subComments != nil {
				comments = []interface{}{subComments}
			}

			return []interface{}{value}, comments
		} else {
			// Returning a map

			comments := make(map[string]interface{})
			if subComments != nil {
				comments[ChangeMeTag] = subComments
			}

			return map[string]interface{}{ChangeMeTag: value}, comments
		}
	}

	// Fall through to property types
	return b.newPropertyType(resourceType, pSpec.Type)
}

func (b Builder) newPrimitive(primitiveType string) interface{} {
	switch primitiveType {
	case "String":
		return ChangeMeTag
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
		panic("PRIMITIVE NOT IMPLEMENTED: " + primitiveType)
	}
}

func (b Builder) newPropertyType(resourceType, propertyType string) (interface{}, interface{}) {
	var ptSpec models.PropertyType
	var ok bool

	ptSpec, ok = b.Spec.PropertyTypes[propertyType]
	if !ok {
		ptSpec, ok = b.Spec.PropertyTypes[resourceType+"."+propertyType]
	}
	if !ok {
		panic("PTYPE NOT IMPLEMENTED: " + resourceType + "." + propertyType)
	}

	// Deal with the case that a property type is directly a plain property
	// for example AWS::Glue::SecurityConfiguration.S3Encryptions
	if ptSpec.Property != emptyProp {
		return b.newProperty(resourceType, ptSpec.Property)
	}

	comments := make(map[string]interface{})

	// Generate properties
	properties := make(map[string]interface{})
	for name, pSpec := range ptSpec.Properties {
		if !pSpec.Required {
			comments[name] = OptionalTag
		}

		if b.BuildIamPolicies && (name == PolicyDocument || name == AssumeRolePolicyDocument) {
			properties[name], comments[name] = iamBuilder.Policy()
		} else if pSpec.Type == propertyType || pSpec.ItemType == propertyType {
			properties[name] = make(map[string]interface{})
		} else {
			properties[name], _ = b.newProperty(resourceType, pSpec)
		}
	}

	return properties, comments
}
