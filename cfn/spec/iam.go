package spec

import "github.com/aws-cloudformation/rain/cfn/spec/models"

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
				"Action": models.Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"Resource": models.Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"NotResource": models.Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"Sid": models.Property{
					PrimitiveType: "String",
				},
				"Principal": models.Property{
					PrimitiveItemType: "String",
					Type:              "Map",
				},
				"NotPrincipal": models.Property{
					PrimitiveItemType: "String",
					Type:              "Map",
				},
				"Effect": models.Property{
					PrimitiveType: "String",
				},
				"NotAction": models.Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"Condition": models.Property{
					PrimitiveItemType: "Json",
					Type:              "Map",
				},
			},
		},
	},
}
