package parse_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/google/go-cmp/cmp"
)

var testFile = "test.yaml"

var testTemplate string

var expected, _ = parse.Map(map[string]interface{}{
	"Parameters": map[string]interface{}{
		"Int": map[string]interface{}{
			"Type":    "Number",
			"Default": int(500000000),
		},
		"Float": map[string]interface{}{
			"Type":    "Number",
			"Default": float64(12345.6789),
		},
		"AccountID": map[string]interface{}{
			"Type":    "String",
			"Default": "0123456789",
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
				"Tags": []interface{}{
					map[string]interface{}{
						"Key": "Empty",
						"Value": map[string]interface{}{
							"Fn::Sub": "",
						},
					},
				},
			},
		},
		"ExecutionRole": map[string]interface{}{
			"Properties": map[string]interface{}{
				"AssumeRolePolicyDocument": map[string]interface{}{
					"Statement": []interface{}{
						map[string]interface{}{
							"Action":    "sts:AssumeRole",
							"Effect":    "Allow",
							"Principal": map[string]interface{}{"Service": "lambda.amazonaws.com"},
						},
					},
					"Version": "2012-10-17",
				},
				"Path": "/",
			},
			"Type": "AWS::IAM::Role",
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
	"AWSTemplateFormatVersion": "2010-09-09",
})

func init() {
	data, err := os.ReadFile(testFile)
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
	source, err := parse.Map(map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": map[string]interface{}{
				"Fn::GetAtt": []interface{}{
					"foo",
					"bar",
				},
			},
			"baz": map[string]interface{}{
				"Fn::GetAtt": []interface{}{
					"foo",
					"bar",
				},
			},
			"quux": []interface{}{
				"mooz",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	goodCase := "foo:\n  bar: !GetAtt foo.bar\n  baz: !GetAtt\n    - foo\n    - bar\n  quux:\n    - mooz"
	badCase := "foo:\n  bar: baz\n  quux: mooz"

	err = parse.Verify(source, goodCase)
	if err != nil {
		t.Error(err)
	}

	err = parse.Verify(source, badCase)
	if err == nil {
		t.Errorf("Verify did not pick up a difference!")
	}
}

func TestEmptySub(t *testing.T) {
	expected, err := parse.Map(map[string]interface{}{
		"Foo": map[string]interface{}{
			"Fn::Sub": "",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	actual, err := parse.String("Foo: !Sub \"\"")
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(actual.Map(), expected.Map()); diff != "" {
		t.Errorf(diff)
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
