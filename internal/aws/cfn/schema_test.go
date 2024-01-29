package cfn_test

import (
	"os"
	"testing"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
)

func TestSchema(t *testing.T) {
	source := `
{
    "typeName": "Rain::Test::Testing",
    "description": "The description",
    "sourceUrl": "https://github.com/aws-cloudformation/rain.git",
    "definitions": { 
		"DefA": {
			"type": "string",
			"enum": [ "Foo", "Bar" ]
		},
		"DefB": {
			"type": "string"
		}
	},
    "properties": {
        "BucketName": {
            "description": "The name of the bucket",
            "type": "string"
        },
		"PropA": {
			"$ref": "#/definitions/DefA"
		}, 
		"PropB": {
			"type": "object", 
			"oneOf": [
				{
					"$ref": "#/definitions/DefA"
				},
				{
					"$ref": "#/definitions/DefB"
				}
			]
		}			
    },
    "additionalProperties": false,
    "tagging": {
        "taggable": false
    },
    "required": [
        "BucketName"
    ],
    "createOnlyProperties": [
        "/properties/BucketName"
    ],
    "primaryIdentifier": [
        "/properties/BucketName"
    ],
    "handlers": {
        "create": {
            "permissions": [
                "s3:ListBucket",
                "s3:GetBucketTagging",
                "s3:PutBucketTagging"
            ]
        },
        "read": {
            "permissions": [
                "s3:ListBucket",
                "s3:GetBucketTagging"
            ]
        },
        "delete": {
            "permissions": [
                "s3:DeleteObject",
                "s3:ListBucket",
                "s3:ListBucketVersions",
                "s3:GetBucketTagging",
                "s3:PutBucketTagging"
            ]
        }
    }
}

`
	s, err := cfn.ParseSchema(source)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := s.Handlers["create"]; ok == false {
		t.Fatalf("handlers missing create")
	}

}

func TestSchemaFiles(t *testing.T) {
	paths := []string{
		"../../../test/schemas/aws-s3-bucket.json",
		"../../../test/schemas/aws-lambda-function.json",
	}

	for _, path := range paths {

		source, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		_, err = cfn.ParseSchema(string(source))
		if err != nil {
			t.Fatal(err)
		}
	}
}
