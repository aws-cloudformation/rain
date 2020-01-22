package format_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/google/go-cmp/cmp"
)

var input = map[string]interface{}{
	"Outputs": map[string]interface{}{
		"Cake": map[string]interface{}{
			"Value": "Lie",
		},
	},
	"Resources": map[string]interface{}{
		"Bucket": map[string]interface{}{
			"Properties": map[string]interface{}{
				"BucketName": map[string]interface{}{
					"Ref": "Name",
				},
			},
			"Type": "AWS::S3::Bucket",
		},
	},
	"Parameters": map[string]interface{}{
		"Name": map[string]interface{}{
			"Type": "String",
		},
	},
}

func TestEncoderWithComments(t *testing.T) {
	options := format.Options{
		Comments: map[string]interface{}{
			"": "This is a thing",
			"Resources": map[string]interface{}{
				"Bucket": map[string]interface{}{
					"": "My bucket",
					"Properties": map[string]interface{}{
						"BucketName": "The name of the bucket",
					},
				},
			},
			"Outputs": "Outputs from resources",
		},
	}

	expected := `# This is a thing
Parameters:
  Name:
    Type: String

Resources:
  Bucket:  # My bucket
    Type: "AWS::S3::Bucket"
    Properties:
      BucketName: !Ref Name  # The name of the bucket

Outputs:  # Outputs from resources
  Cake:
    Value: Lie`

	actual := format.Anything(input, options)

	if d := cmp.Diff(actual, expected); d != "" {
		t.Errorf(d)
	}
}

func BenchmarkJson(b *testing.B) {
	for n := 0; n < b.N; n++ {
		format.Anything(input, format.Options{
			Style: format.JSON,
		})
	}
}

func BenchmarkYaml(b *testing.B) {
	for n := 0; n < b.N; n++ {
		format.Anything(input, format.Options{
			Style: format.YAML,
		})
	}
}
