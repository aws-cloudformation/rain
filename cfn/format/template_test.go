package format_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/aws-cloudformation/rain/cfn/parse"
)

func TestTemplate(t *testing.T) {
	template, err := parse.String(`
Outputs:
  Bucket:
    Value: !Ref Bucket2
Resources:
  Bucket1:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Sub ${Bucket2}-newer
  Bucket2:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Ref Name
Parameters:
  Name:
    Type: String
`)

	expected := `Parameters:
  Name:
    Type: String

Resources:
  Bucket2:
    Type: "AWS::S3::Bucket"
    Properties:
      BucketName: !Ref Name

  Bucket1:
    Type: "AWS::S3::Bucket"
    Properties:
      BucketName: !Sub ${Bucket2}-newer

Outputs:
  Bucket:
    Value: !Ref Bucket2`

	if err != nil {
		t.Error(err)
	}

	actual := format.Template(template, format.Options{})

	if actual != expected {
		t.Errorf("Got:\n%s\nWant:\n%s\n", actual, expected)
	}
}
