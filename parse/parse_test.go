package parse

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var testFile = "test.yaml"

var testTemplate string

var expected = map[string]interface{}{
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
}

func init() {
	data, err := ioutil.ReadFile(testFile)
	if err != nil {
		panic(err)
	}

	testTemplate = string(data)
}

func TestRead(t *testing.T) {
	actual, err := Read(strings.NewReader(testTemplate))
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf(diff)
	}
}

func TestReadFile(t *testing.T) {
	actual, err := ReadFile(testFile)
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf(diff)
	}
}

func TestReadString(t *testing.T) {
	actual, err := ReadString(testTemplate)
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf(diff)
	}
}

func TestVerifyOutput(t *testing.T) {
	source := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": "baz",
			"quux": []interface{}{
				"mooz",
			},
		},
	}

	goodCase := "foo:\n  bar: baz\n  quux:\n    - mooz"
	badCase := "foo:\n  bar: baz\n  quux: mooz"

	var err error

	err = VerifyOutput(source, goodCase)
	if err != nil {
		t.Error(err)
	}

	err = VerifyOutput(source, badCase)
	if err == nil {
		t.Errorf("Verify did not pick up a difference!")
	}
}
