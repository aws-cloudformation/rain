package spec

// Iam is generated from the specification file
var Iam = Spec{
	ResourceSpecificationVersion: "1.0.0",
	PropertyTypes: map[string]*PropertyType{
		"Policy": {
			Properties: map[string]*Property{
				"Id": {
					PrimitiveType: "String",
				},
				"Statement": {
					ItemType: "Statement",
					Type:     "List",
				},
				"Version": {
					PrimitiveType: "String",
				},
			},
		},
		"Statement": {
			Properties: map[string]*Property{
				"Principal": {
					PrimitiveItemType: "String",
					Type:              "Map",
				},
				"Condition": {
					PrimitiveItemType: "Json",
					Type:              "Map",
				},
				"NotAction": {
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"NotPrincipal": {
					PrimitiveItemType: "String",
					Type:              "Map",
				},
				"Resource": {
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"Sid": {
					PrimitiveType: "String",
				},
				"Action": {
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"Effect": {
					PrimitiveType: "String",
				},
				"NotResource": {
					PrimitiveItemType: "String",
					Type:              "List",
				},
			},
		},
	},
}
