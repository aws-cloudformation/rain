package spec

import "github.com/aws-cloudformation/rain/cfn/spec/models"

// Iam is generated from the specification file
var Iam = models.Spec{
	ResourceSpecificationVersion: "1.0.0",
	PropertyTypes: map[string]models.PropertyType{
		"Statement": models.PropertyType{
			Properties: map[string]models.Property{
				"Condition": models.Property{
					PrimitiveItemType: "Json",
					Type:              "Map",
				},
				"NotPrincipal": models.Property{
					PrimitiveItemType: "String",
					Type:              "Map",
				},
				"Principal": models.Property{
					PrimitiveItemType: "String",
					Type:              "Map",
				},
				"Sid": models.Property{
					PrimitiveType: "String",
				},
				"Action": models.Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"Effect": models.Property{
					PrimitiveType: "String",
				},
				"NotAction": models.Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"NotResource": models.Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"Resource": models.Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
			},
		},
		"Policy": models.PropertyType{
			Properties: map[string]models.Property{
				"Id": models.Property{
					PrimitiveType: "String",
				},
				"Statement": models.Property{
					ItemType: "Statement",
					Type:     "List",
				},
				"Version": models.Property{
					PrimitiveType: "String",
				},
			},
		},
	},
}
