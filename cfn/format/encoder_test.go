package format_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/cfn/format"
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
