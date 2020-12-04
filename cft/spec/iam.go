package spec

// Iam is generated from the specification file
var Iam = Spec{
	PropertyTypes: map[string]*PropertyType{
		"Policy": &PropertyType{
			Properties: map[string]*Property{
				"Id": &Property{
					PrimitiveType: "String",
				},
				"Statement": &Property{
					ItemType: "Statement",
					Type:     "List",
				},
				"Version": &Property{
					PrimitiveType: "String",
				},
			},
		},
		"Statement": &PropertyType{
			Properties: map[string]*Property{
				"Action": &Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"Condition": &Property{
					PrimitiveItemType: "Json",
					Type:              "Map",
				},
				"Effect": &Property{
					PrimitiveType: "String",
				},
				"NotAction": &Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"NotPrincipal": &Property{
					PrimitiveItemType: "String",
					Type:              "Map",
				},
				"NotResource": &Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"Principal": &Property{
					PrimitiveItemType: "String",
					Type:              "Map",
				},
				"Resource": &Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"Sid": &Property{
					PrimitiveType: "String",
				},
			},
		},
	},
	ResourceSpecificationVersion: "1.0.0",
}
