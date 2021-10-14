package spec

var Iam = map[string]map[string]interface{}{
	"Policy": map[string]interface{}{
		"additionalProperties": false,
		"definitions": map[string]interface{}{
			"Statement": map[string]interface{}{
				"$schema":              "http://json-schema.org/draft-07/schema#",
				"additionalProperties": false,
				"type":                 "object",
				"properties": map[string]interface{}{
					"Sid": map[string]interface{}{
						"type": "string",
					},
					"Principal": map[string]interface{}{
						"type": "object",
						"additionalProperties": map[string]interface{}{
							"type": "string",
						},
					},
					"NotPrincipal": map[string]interface{}{
						"type": "object",
						"additionalProperties": map[string]interface{}{
							"type": "string",
						},
					},
					"Effect": map[string]interface{}{
						"type": "string",
					},
					"Action": map[string]interface{}{
						"type": "array",
						"item": map[string]interface{}{
							"type": "string",
						},
					},
					"NotAction": map[string]interface{}{
						"type": "array",
						"item": map[string]interface{}{
							"type": "string",
						},
					},
					"Resource": map[string]interface{}{
						"type": "array",
						"item": map[string]interface{}{
							"type": "string",
						},
					},
					"NotResource": map[string]interface{}{
						"type": "array",
						"item": map[string]interface{}{
							"type": "string",
						},
					},
					"Condition": map[string]interface{}{
						"type": "object",
					},
				},
			},
		},
		"description": "IAM Policy",
		"properties": map[string]interface{}{
			"Version": map[string]interface{}{
				"type": "string",
			},
			"Id": map[string]interface{}{
				"type": "string",
			},
			"Statement": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"$ref": "#/definitions/Statement",
				},
			},
		},
	},
}
