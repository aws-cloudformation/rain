// In this file, we convert a version of the SAM spec that has been simplified to the
// old CloudFormation spec format into the new registry schema format. This
// allows us to build boilerplate templates for SAM resource types using the same
// code that we use for normal registry resource types.
// Ideally we would parse the actual SAM spec, but the file is huge and complex,
// and SAM transforms are relatively stable, so we shouldn't have issues with
// the output being out of date.
package build

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/config"
)

// SamProp represents the old cfn spec types, which are... odd
type SamProp struct {
	Documentation               string
	Required                    bool
	PrimitiveType               string
	PrimitiveItemType           string
	UpdateType                  string
	PrimitiveTypes              []string
	Type                        string
	Types                       []string
	ItemType                    string
	InclusivePrimitiveItemTypes []string
	InclusiveItemTypes          []string
}

type SamType struct {
	Documentation string
	Properties    map[string]*SamProp
}

type SamSpec struct {
	ResourceSpecificationVersion   string
	ResourceSpecificationTransform string
	Globals                        any
	ResourceTypes                  map[string]*SamType
	PropertyTypes                  map[string]*SamType
}

func convertPrimitiveType(t string) string {

	switch t {
	case "String":
		return "string"
	case "Integer":
		return "integer"
	case "Json":
		return "object"
	case "Boolean":
		return "boolean"
	case "Double":
		return "number"
	case "Map":
		return "object"
	default:
		return ""
	}
}

func makeRef(name string) string {
	return "#/definitions/" + name
}

// convertSAMProp translates a SAM (old Cfn spec) type to a registry schema type
func convertSAMProp(samType *SamType) (*cfn.Prop, error) {

	retval := &cfn.Prop{}
	retval.Properties = make(map[string]*cfn.Prop)

	for samPropName, samProp := range samType.Properties {
		cfnProp := &cfn.Prop{}
		switch {
		case samProp.Type != "":
			switch samProp.Type {
			case "List":
				cfnProp.Type = "array"
				if samProp.PrimitiveItemType != "" {
					pt := convertPrimitiveType(samProp.PrimitiveItemType)
					if pt == "" {
						return nil, fmt.Errorf("unexpected PrimitiveItemType: %s %s", samPropName, samProp.PrimitiveItemType)
					}
					cfnProp.Items = &cfn.Prop{Type: pt}
				} else if samProp.ItemType != "" {
					cfnProp.Items = &cfn.Prop{Type: "object", Ref: makeRef(samProp.ItemType)}
				} else if len(samProp.InclusivePrimitiveItemTypes) > 0 {
					ipt := convertPrimitiveType(samProp.InclusivePrimitiveItemTypes[0])
					if ipt == "" {
						return nil, fmt.Errorf("unexpected InclusivePrimitiveItemTypes for %s: %s", samPropName, samProp.InclusivePrimitiveItemTypes[0])
					}
					cfnProp.Items = &cfn.Prop{Type: ipt}
				} else {
					return nil, fmt.Errorf("expected %s to have ItemType or PrimitiveItemType", samPropName)
				}
			case "Map":
				cfnProp.Type = "object"
				if samProp.ItemType != "" {
					cfnProp.Ref = makeRef(samProp.ItemType)
				} else {
					cfnProp.Ref = convertPrimitiveType(samProp.PrimitiveItemType)
				}
			default:
				cfnProp.Type = "object"
				cfnProp.Ref = makeRef(samProp.Type)
			}
		case samProp.PrimitiveType != "":
			cfnProp.Type = convertPrimitiveType(samProp.PrimitiveType)
			if cfnProp.Type == "" {
				return nil, fmt.Errorf("unexpected PrimitiveType: %s.%s", samPropName, samProp.PrimitiveType)
			}
		case len(samProp.PrimitiveTypes) > 0:
			pt := convertPrimitiveType(samProp.PrimitiveTypes[0])
			if pt == "" {
				return nil, fmt.Errorf("unexpected PrimitiveTypes: %s.%s", samPropName, samProp.PrimitiveTypes[0])
			}
			if len(samProp.Types) > 0 {
				// This is one of those odd ones that can be a primitive or an object
				// This is not something that normal resource types do
				// We'll just keep it simple and output the primitive
				cfnProp.Type = pt
			} else {
				cfnProp.Type = "array"
				cfnProp.Items = &cfn.Prop{}
				cfnProp.Items.Type = pt
			}
		default:
			cfnProp.Type = "object"
		}
		if cfnProp.Type == "" {
			config.Debugf("Missing Type for %s", samPropName)
			cfnProp.Type = "object"
		}
		retval.Properties[samPropName] = cfnProp
	}
	return retval, nil
}

// Get the transform type information from the SAM spec and convert it
func convertSAMSpec(source string, typeName string) (*cfn.Schema, error) {
	var spec SamSpec
	err := json.Unmarshal([]byte(source), &spec)
	if err != nil {
		return nil, err
	}

	schema := &cfn.Schema{}

	schema.TypeName = typeName
	samType, found := spec.ResourceTypes[typeName]
	if !found {
		return nil, fmt.Errorf("sam type not found in spec: %s", typeName)
	}
	schema.Description = samType.Documentation

	// Definitions
	schema.Definitions = make(map[string]*cfn.Prop)
	for k, v := range spec.PropertyTypes {
		if strings.HasPrefix(k, typeName) {
			propName := strings.Replace(k, typeName+".", "", 1)
			def, err := convertSAMProp(v)
			def.Type = "object"
			if err != nil {
				return nil, err
			}
			schema.Definitions[propName] = def
		}
	}

	// Properties
	schema.Properties = make(map[string]*cfn.Prop)
	found = false
	for k, v := range spec.ResourceTypes {
		config.Debugf("SAM resource: %s", k)
		if k == typeName {
			found = true
			p, err := convertSAMProp(v)
			if err != nil {
				return nil, err
			}
			schema.Properties = p.Properties
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("did not find %s in the SAM spec", typeName)
	}

	return schema, nil
}
