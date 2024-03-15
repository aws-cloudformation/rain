package cfn

import (
	"encoding/json"
	"reflect"

	"github.com/aws-cloudformation/rain/internal/config"
)

type SchemaLike interface {
	GetRequired() []string
	GetProperties() map[string]*Prop
}

// Represents a registry schema property or definition
type Prop struct {
	Description          string           `json:"description"`
	Items                *Prop            `json:"items"`
	Type                 any              `json:"type"`
	UniqueItems          bool             `json:"uniqueItems"`
	InsertionOrder       bool             `json:"insertionOrder"`
	Ref                  string           `json:"$ref"`
	MaxLength            int              `json:"maxLength"`
	MinLength            int              `json:"minLength"`
	Pattern              string           `json:"pattern"`
	Examples             []string         `json:"examples"`
	AdditionalProperties bool             `json:"additionalProperties"`
	Properties           map[string]*Prop `json:"properties"`
	Enum                 []any            `json:"enum"`
	Required             []string         `json:"required"`
	OneOf                []*Prop          `json:"oneOf"`
	AnyOf                []*Prop          `json:"anyOf"`
}

func (p *Prop) GetProperties() map[string]*Prop {
	return p.Properties
}

func (p *Prop) GetRequired() []string {
	return p.Required
}

// Represents a registry schema for a resource type like AWS::S3::Bucket
type Schema struct {
	TypeName             string           `json:"typeName"`
	Description          string           `json:"description"`
	SourceUrl            string           `json:"sourceUrl"`
	Definitions          map[string]*Prop `json:"definitions"`
	Handlers             map[string]any   `json:"handlers"`
	PrimaryIdentifier    []string         `json:"primaryIdentifier"`
	Properties           map[string]*Prop `json:"properties"`
	AdditionalProperties bool             `json:"additionalProperties"`
	Tagging              map[string]any   `json:"tagging"`
	Required             []string         `json:"required"`
	ReadOnlyProperties   []string         `json:"readOnlyProperties"`
	WriteOnlyProperties  []string         `json:"writeOnlyProperties"`
	CreateOnlyProperties []string         `json:"createOnlyProperties"`
}

func (s *Schema) GetProperties() map[string]*Prop {
	return s.Properties
}

func (s *Schema) GetRequired() []string {
	return s.Required
}

// ParseSchema unmarshals the text of a registry schema into a struct
func ParseSchema(source string) (*Schema, error) {
	var s Schema
	err := json.Unmarshal([]byte(source), &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Patch applies patches to the schema to add things like undocumented enums
func (schema *Schema) Patch() error {
	config.Debugf("Patching %s", schema.TypeName)
	switch schema.TypeName {
	case "AWS::Lightsail::Instance":
		return patchLightsailInstance(schema)
	case "AWS::Lightsail::Bucket":
		return patchLightsailBucket(schema)
	case "AWS::Lightsail::Database":
		return patchLightsailDatabase(schema)
	case "AWS::Lightsail::Alarm":
		return patchLightsailAlarm(schema)
	case "AWS::Lightsail::Distribution":
		return patchLightsailDistribution(schema)
	case "AWS::SES::ConfigurationSetEventDestination":
		return patchSESConfigurationSetEventDestination(schema)
	case "AWS::SES::ContactList":
		return patchSESContactList(schema)
	case "AWS::IAM::Role":
		return patchIAMRole(schema)

	}
	return nil
}

func ConvertPropType(t any) any {
	if t == nil {
		return ""
	}
	rt := reflect.TypeOf(t)
	switch rt.Kind() {
	case reflect.Slice:
		fallthrough
	case reflect.Array:
		// Things like PolicyDocument are [string, object]
		return "object"
	}
	return t
}
