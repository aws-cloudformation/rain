package models

import "strings"

const (
	TypeEmpty = ""
	TypeMap   = "Map"
	TypeList  = "List"
)

// Spec is a representation of the CloudFormation specification document
type Spec struct {
	ResourceSpecificationVersion string
	PropertyTypes                map[string]PropertyType
	ResourceTypes                map[string]ResourceType
}

func (s Spec) String() string {
	return formatStruct(s)
}

// PropertyType represents a propertytype node
// in the CloudFormation specification
type PropertyType struct {
	Property
	Documentation string
	Properties    map[string]Property
}

// ResourceType represents a resourcetype node
// in the CloudFormation specification
type ResourceType struct {
	Attributes           map[string]Attribute
	Documentation        string
	Properties           map[string]Property
	AdditionalProperties bool
}

type Property struct {
	Documentation     string
	DuplicatesAllowed bool
	ItemType          string
	PrimitiveItemType string
	PrimitiveType     string
	Required          bool
	Type              string
	UpdateType        string
}

type Attribute struct {
	ItemType          string
	PrimitiveItemType string
	PrimitiveType     string
	Type              string
}

func (p Property) TypeName() string {
	if p.PrimitiveType != TypeEmpty {
		if p.PrimitiveType == TypeList || p.PrimitiveType == TypeMap {
			if p.PrimitiveItemType != "" {
				return p.PrimitiveType + "/" + p.PrimitiveItemType
			}

			return p.PrimitiveType + "/" + p.ItemType
		}

		return p.PrimitiveType
	}

	return p.Type
}

// ResolveResource returns a list of possible Resource names for
// the provided suffix
func (s Spec) ResolveResource(suffix string) []string {
	options := make([]string, 0)

	for r, _ := range s.ResourceTypes {
		if strings.HasSuffix(r, suffix) {
			options = append(options, r)
		}
	}

	return options
}
