package spec

import "github.com/aws-cloudformation/rain/cfn/spec/models"

// Iam is generated from the specification file
var Iam = models.Spec{
	ResourceSpecificationVersion: "1.0.0",
	PropertyTypes: map[string]models.PropertyType{
		"Policy": models.PropertyType{
			Properties: map[string]models.Property{
				"Version": models.Property{
					PrimitiveType: "String",
				},
				"Id": models.Property{
					PrimitiveType: "String",
				},
				"Statement": models.Property{
					ItemType: "Statement",
					Type:     "List",
				},
			},
		},
		"Statement": models.PropertyType{
			Properties: map[string]models.Property{
				"Effect": models.Property{
					PrimitiveType: "String",
				},
				"Resource": models.Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"Action": models.Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"NotAction": models.Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"NotPrincipal": models.Property{
					PrimitiveItemType: "String",
					Type:              "Map",
				},
				"NotResource": models.Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"Principal": models.Property{
					PrimitiveItemType: "String",
					Type:              "Map",
				},
				"Sid": models.Property{
					PrimitiveType: "String",
				},
				"Condition": models.Property{
					PrimitiveItemType: "Json",
					Type:              "Map",
				},
			},
		},
	},
}
