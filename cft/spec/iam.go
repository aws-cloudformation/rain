package spec

var Iam = map[string]map[string]interface{}{
	"Policy": map[string]interface{}{
		"additionalProperties": false,
		"definitions": map[string]interface{}{
			"Statement": {
				"additionalProperties": false,
				"type":                 "object",
				"properties": {
					"Sid": {
						"type": "string",
					},
					"Principal": {
						"type": "object",
						"additionalProperties": {
							"type": "string",
						},
					},
					"NotPrincipal": {
						"type": "object",
						"additionalProperties": {
							"type": "string",
						},
					},
					"Effect": {
						"type": "string",
					},
					"Action": {
						"type": "array",
						"item": {
							"type": "string",
						},
					},
					"NotAction": {
						"type": "array",
						"item": {
							"type": "string",
						},
					},
					"Resource": {
						"type": "array",
						"item": {
							"type": "string",
						},
					},
					"NotResource": {
						"type": "array",
						"item": {
							"type": "string",
						},
					},
					"Condition": {
						"type": "object",
					},
				},
			},
		},
		"description": "IAM Policy",
		"properties": map[string]interface{}{
			"Version": {
				"type": "string",
			},
			"Id": {
				"type": "string",
			},
			"Statement": {
				"type": "array",
				"items": {
					"$ref": "#/definitions/Statement",
				},
			},
		},
	},
}
