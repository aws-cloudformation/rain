package build

import (
	"encoding/json"
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
)

type SamProp struct {
	Documentation               string
	Required                    bool
	PrimitiveType               string
	UpdateType                  string
	PrimitiveTypes              []string
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
	// TODO: Convert from spec.PropertyTypes
	schema.Definitions = make(map[string]*cfn.Prop)

	// Properties
	// TODO
	schema.Properties = make(map[string]*cfn.Prop)

	return schema, nil
}
