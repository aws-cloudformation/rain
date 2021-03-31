package cft_test

import (
	"fmt"
	"testing"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func toNode(in interface{}) *yaml.Node {
	n := &yaml.Node{}

	err := n.Encode(in)
	if err != nil {
		panic(err)
	}

	return n
}

func get(in interface{}, path []interface{}) interface{} {
	if len(path) == 0 {
		return in
	}

	head, tail := path[0], path[1:]

	switch v := head.(type) {
	case string:
		return get(in.(map[string]interface{})[v], tail)
	case int:
		return get(in.([]interface{})[v], tail)
	default:
		panic(fmt.Errorf("Unexpected path entry: %#v", head))
	}
}

func TestMatchPath(t *testing.T) {
	tplMap := map[string]interface{}{
		"Parameters": map[string]interface{}{
			"BucketName": map[string]interface{}{
				"Type": "String",
			},
		},
		"Resources": map[string]interface{}{
			"Bucket": map[string]interface{}{
				"Type": "AWS::S3::Bucket",
				"Properties": map[string]interface{}{
					"BucketName": map[string]interface{}{
						"Ref": "BucketName",
					},
				},
			},
			"Queue": map[string]interface{}{
				"Type": "AWS::SQS::Queue",
				"Properties": map[string]interface{}{
					"QueueName": map[string]interface{}{
						"Ref": "BucketName",
					},
				},
				"Tags": []interface{}{
					map[string]interface{}{
						"Key":   "First",
						"Value": "1",
					},
					map[string]interface{}{
						"Key":   "Second",
						"Value": "2",
					},
				},
			},
		},
		"Outputs": map[string]interface{}{
			"BucketName": map[string]interface{}{
				"Value": map[string]interface{}{
					"Ref": "Bucket",
				},
			},
		},
	}

	tpl, _ := parse.Map(tplMap)

	testCases := []struct {
		path     string
		expected []*yaml.Node
	}{
		{path: "Parameters/BucketName", expected: []*yaml.Node{
			toNode(get(tplMap, []interface{}{"Parameters", "BucketName"})),
		}},
		{path: "Resources/*", expected: []*yaml.Node{
			toNode(get(tplMap, []interface{}{"Resources", "Bucket"})),
			toNode(get(tplMap, []interface{}{"Resources", "Queue"})),
		}},
		{path: "Resources/*/Tags/0", expected: []*yaml.Node{
			toNode(get(tplMap, []interface{}{"Resources", "Queue", "Tags", 0})),
		}},
		{path: "Resources/*/Properties", expected: []*yaml.Node{
			toNode(get(tplMap, []interface{}{"Resources", "Bucket", "Properties"})),
			toNode(get(tplMap, []interface{}{"Resources", "Queue", "Properties"})),
		}},
		{path: "Resources/*|Type==AWS::S3::Bucket/Properties", expected: []*yaml.Node{
			toNode(get(tplMap, []interface{}{"Resources", "Bucket", "Properties"})),
		}},
		{path: "Resources/*/Type", expected: []*yaml.Node{
			toNode(get(tplMap, []interface{}{"Resources", "Bucket", "Type"})),
			toNode(get(tplMap, []interface{}{"Resources", "Queue", "Type"})),
		}},
		{path: "**/Type", expected: []*yaml.Node{
			toNode(get(tplMap, []interface{}{"Parameters", "BucketName", "Type"})),
			toNode(get(tplMap, []interface{}{"Resources", "Bucket", "Type"})),
			toNode(get(tplMap, []interface{}{"Resources", "Queue", "Type"})),
		}},
		{path: "**/*|Ref", expected: []*yaml.Node{
			toNode(get(tplMap, []interface{}{"Outputs", "BucketName", "Value"})),
			toNode(get(tplMap, []interface{}{"Resources", "Bucket", "Properties", "BucketName"})),
			toNode(get(tplMap, []interface{}{"Resources", "Queue", "Properties", "QueueName"})),
		}},
		{path: "**/Tags/*|Key==Second/*", expected: []*yaml.Node{
			toNode(get(tplMap, []interface{}{"Resources", "Queue", "Tags", 1, "Key"})),
			toNode(get(tplMap, []interface{}{"Resources", "Queue", "Tags", 1, "Value"})),
		}},
		{path: "**/*|Tags", expected: []*yaml.Node{
			toNode(get(tplMap, []interface{}{"Resources", "Queue"})),
		}},
	}

	for _, testCase := range testCases {
		results := make([]*yaml.Node, 0)
		for n := range tpl.MatchPath(testCase.path) {
			results = append(results, n)
		}

		if len(results) != len(testCase.expected) {
			t.Errorf("%s: Expected %d results, got %d", testCase.path, len(testCase.expected), len(results))
		}

		for i, actual := range results {
			expected := testCase.expected[i]

			if d := cmp.Diff(expected, actual); d != "" {
				t.Error(d)
			}
		}
	}
}
