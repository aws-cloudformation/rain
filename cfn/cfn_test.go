package cfn_test

import (
	"reflect"
	"testing"

	"github.com/aws-cloudformation/rain/cfn"
)

var testCase = cfn.New(map[string]interface{}{
	"Parameters": map[string]interface{}{
		"Name": map[string]interface{}{
			"Type": "String",
		},
	},
	"Resources": map[string]interface{}{
		"Bucket": map[string]interface{}{
			"Type": "AWS::S3::Bucket",
			"Properties": map[string]interface{}{
				"BucketName": map[string]interface{}{
					"Ref": "Name",
				},
			},
		},
	},
	"Outputs": map[string]interface{}{
		"Bucket": map[string]interface{}{
			"Value": map[string]interface{}{
				"Ref": "Bucket",
			},
		},
	},
})

func TestGraph(t *testing.T) {
	graph := testCase.Graph()

	actual := graph.Nodes()
	expected := []interface{}{
		cfn.Element{"Name", "Parameters"},
		cfn.Element{"Bucket", "Resources"},
		cfn.Element{"Bucket", "Outputs"},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Template graph is wrong:\n%#v\n!=\n%#v\n", expected, actual)
	}
}
