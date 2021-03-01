// Package spec contains generated models for CloudFormation and IAM
package spec

//go:generate bash -c "internal/sam.sh >internal/SamSpecification.yaml"
//go:generate go run internal/main.go

import "strings"

const (
	// TypeEmpty flags an empty type
	TypeEmpty = ""

	// TypeMap flags a map type
	TypeMap = "Map"

	// TypeList flags a list type
	TypeList = "List"
)

// Spec is a representation of the CloudFormation specification document
type Spec struct {
	ResourceSpecificationVersion string
	PropertyTypes                map[string]*PropertyType
	ResourceTypes                map[string]*ResourceType
}

func (s Spec) String() string {
	return formatStruct(s)
}

// PropertyType represents a propertytype node
// in the CloudFormation specification
type PropertyType struct {
	Property
	Documentation string
	Properties    map[string]*Property
}

// ResourceType represents a resourcetype node
// in the CloudFormation specification
type ResourceType struct {
	Attributes           map[string]*Attribute
	Documentation        string
	Properties           map[string]*Property
	AdditionalProperties bool
}

// Property represents a property within a spec
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

// Attribute represents an attribute of a type
type Attribute struct {
	ItemType          string
	DuplicatesAllowed bool
	PrimitiveItemType string
	PrimitiveType     string
	Type              string
}

// TypeName returns the Attribute's name
func (a Attribute) TypeName() string {
	if a.PrimitiveType != TypeEmpty {
		if a.PrimitiveType == TypeList || a.PrimitiveType == TypeMap {
			if a.PrimitiveItemType != "" {
				return a.PrimitiveType + "/" + a.PrimitiveItemType
			}

			return a.PrimitiveType + "/" + a.ItemType
		}

		return a.PrimitiveType
	}

	return a.Type
}

// TypeName returns the Property's name
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
	suffix = strings.ToLower(suffix)

	options := make([]string, 0)

	for r := range s.ResourceTypes {
		if strings.HasSuffix(strings.ToLower(r), suffix) {
			options = append(options, r)
		}
	}

	return options
}
