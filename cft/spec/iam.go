package spec

// Iam is generated from the specification file
var Iam = Spec{
	ResourceSpecificationVersion: "1.0.0",
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
				"Principal": &Property{
					PrimitiveItemType: "String",
					Type:              "Map",
				},
				"Condition": &Property{
					PrimitiveItemType: "Json",
					Type:              "Map",
				},
				"NotAction": &Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"NotPrincipal": &Property{
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
				"Action": &Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
				"Effect": &Property{
					PrimitiveType: "String",
				},
				"NotResource": &Property{
					PrimitiveItemType: "String",
					Type:              "List",
				},
			},
		},
	},
}
