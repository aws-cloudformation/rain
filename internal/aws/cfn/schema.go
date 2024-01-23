package cfn

import "encoding/json"

// Represents a registry schema property or definition
type Prop struct {
	Description          string   `json:"description"`
	Items                *Prop    `json:"items"`
	Type                 string   `json:"type"`
	UniqueItems          bool     `json:"uniqueItems"`
	InsertionOrder       bool     `json:"insertionOrder"`
	Ref                  string   `json:"$ref"`
	MaxLength            int      `json:"maxLength"`
	MinLength            int      `json:"minLength"`
	Pattern              string   `json:"pattern"`
	Examples             []string `json:"examples"`
	AdditionalProperties bool     `json:"additionalProperties"`
	Properties           *Prop    `json:"properties"`
	Enum                 []string `json:"enum"`
	Required             []string `json:"required"`
}

// Represents a registry schema for a resource type like AWS::S3::Bucket
type Schema struct {
	TypeName             string           `json:"typeName"`
	Description          string           `json:"description"`
	SourceUrl            string           `json:"sourceUrl"`
	Definitions          map[string]*Prop `json:"definitions"`
	Handlers             map[string]any   `json:"handlers"`
	PrimaryIdentifier    []string         `json:"primaryIdentifier    "`
	Properties           map[string]*Prop `json:"properties"`
	AdditionalProperties bool             `json:"additionalProperties"`
	Tagging              map[string]any   `json:"tagging"`
	Required             []string         `json:"required"`
	ReadOnlyProperties   []string         `json:"readOnlyProperties"`
	WriteOnlyProperties  []string         `json:"writeOnlyProperties"`
	CreateOnlyProperties []string         `json:"createOnlyProperties"`
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
