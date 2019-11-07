package parse_test

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/aws-cloudformation/rain/cfn/parse"
	"github.com/google/go-cmp/cmp"
)

var testFile = "test.yaml"

var testTemplate string

var expected, _ = parse.Map(map[string]interface{}{
	"Parameters": map[string]interface{}{
		"Number": map[string]interface{}{
			"Type":    "Number",
			"Default": float64(500000000),
		},
	},
	"Resources": map[string]interface{}{
		"Bucket1": map[string]interface{}{
			"Type": "AWS::S3::Bucket",
			"Properties": map[string]interface{}{
				"BucketName": map[string]interface{}{
					"Fn::Base64": map[string]interface{}{
						"Ref": "Cakes",
					},
				},
			},
		},
		"Bucket2": map[string]interface{}{
			"Type": "AWS::S3::Bucket",
			"Properties": map[string]interface{}{
				"BucketName": map[string]interface{}{
					"Fn::Base64": map[string]interface{}{
						"Ref": "Cakes",
					},
				},
			},
		},
	},
	"Outputs": map[string]interface{}{
		"Bucket1Arn": map[string]interface{}{
			"Value": map[string]interface{}{
				"Fn::GetAtt": []interface{}{
					"Bucket1",
					"Arn",
				},
			},
		},
		"Bucket1Name": map[string]interface{}{
			"Value": map[string]interface{}{
				"Ref": "Bucket1",
			},
		},
		"Bucket2Arn": map[string]interface{}{
			"Value": map[string]interface{}{
				"Fn::GetAtt": []interface{}{
					"Bucket2",
					"Arn",
				},
			},
		},
	},
})

func init() {
	data, err := ioutil.ReadFile(testFile)
	if err != nil {
		panic(err)
	}

	testTemplate = string(data)
}

func TestRead(t *testing.T) {
	actual, err := parse.Reader(strings.NewReader(testTemplate))
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(actual.Map(), expected.Map()); diff != "" {
		t.Errorf(diff)
	}
}

func TestReadFile(t *testing.T) {
	actual, err := parse.File(testFile)
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(actual.Map(), expected.Map()); diff != "" {
		t.Errorf(diff)
	}
}

func TestReadString(t *testing.T) {
	actual, err := parse.String(testTemplate)
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(actual.Map(), expected.Map()); diff != "" {
		t.Errorf(diff)
	}
}

func TestVerifyOutput(t *testing.T) {
	source, _ := parse.Map(map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": map[string]interface{}{
				"Fn::GetAtt": []interface{}{
					"foo",
					"bar",
				},
			},
			"baz": map[string]interface{}{
				"Fn::GetAtt": "foo.bar",
			},
			"quux": []interface{}{
				"mooz",
			},
		},
	})

	goodCase := "foo:\n  bar: !GetAtt foo.bar\n  baz: !GetAtt\n    - foo\n    - bar\n  quux:\n    - mooz"
	badCase := "foo:\n  bar: baz\n  quux: mooz"

	var err error

	err = parse.Verify(source, goodCase)
	if err != nil {
		t.Error(err)
	}

	err = parse.Verify(source, badCase)
	if err == nil {
		t.Errorf("Verify did not pick up a difference!")
	}
}

func Example() {
	template, _ := parse.String(`
Resources:
  Bucket:
    Type: AWS::S3::Bucket
`)

	fmt.Println(template.Map())
	// Output:
	// map[Resources:map[Bucket:map[Type:AWS::S3::Bucket]]]
}
