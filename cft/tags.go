package cft

// Tags is a mapping from YAML short tags to full instrincic function names
var Tags = map[string]string{
	"!And":           "Fn::And",
	"!Base64":        "Fn::Base64",
	"!Cidr":          "Fn::Cidr",
	"!Equals":        "Fn::Equals",
	"!FindInMap":     "Fn::FindInMap",
	"!GetAZs":        "Fn::GetAZs",
	"!GetAtt":        "Fn::GetAtt",
	"!If":            "Fn::If",
	"!ImportValue":   "Fn::ImportValue",
	"!Join":          "Fn::Join",
	"!Not":           "Fn::Not",
	"!Or":            "Fn::Or",
	"!Select":        "Fn::Select",
	"!Split":         "Fn::Split",
	"!Sub":           "Fn::Sub",
	"!Ref":           "Ref",
	"!Condition":     "Condition",
	"!Rain::Embed":   "Rain::Embed",
	"!Rain::Include": "Rain::Include",
	"!Rain::S3Http":  "Rain::S3Http",
	"!Rain::S3":      "Rain::S3",
}
