package diff

import (
	"testing"

	"github.com/aws-cloudformation/rain/parse"
)

const a = `Parameters:
  BucketName:
    Type: String
Resources:
  Bucket:
    Type: AWS::S3::Bucket
  Properties:
    BucketName: !Ref BucketName
Outputs:
  BucketArn:
    Value: !GetAtt Bucket.Arn
`

const b = `Parameters:
  BucketName:
    Type: String
Resources:
  Bucket:
    Type: AWS::S3::Bucket
  Properties:
    BucketName:
      Ref: BucketName
Outputs:
  BucketArn:
    Value:
      Fn::GetAtt:
        - Bucket
        - Arn
`

func TestCompareTemplateIntrinsics(t *testing.T) {
	at, err := parse.ReadString(a)
	if err != nil {
		t.Error(err)
	}

	bt, err := parse.ReadString(b)
	if err != nil {
		t.Error(err)
	}

	actual := Compare(at, bt)

	if actual.Mode() != Unchanged {
		t.Errorf("Templates are not equal! %s", Format(actual, false))
	}
}
