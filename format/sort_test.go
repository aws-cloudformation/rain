package format

import (
	"reflect"
	"testing"

	"github.com/aws-cloudformation/rain/parse"
)

var templateString = `
Parameters:
  Type: String
  Name: Name

Outputs:
  Bucket:
    Value: !Ref ZBucket

Resources:
  ABucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Ref ARole

  ZBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Ref Name

  ARole:
    Type: AWS::IAM::Role
    Properties:
      Policies:
      - PolicyName: S3Access
        PolicyDocument:
        Statement:
          - Effect: Allow
            Action: "s3:*"
            Resource:
              - !GetAtt ZBucket.Arn
`

func TestOrdering(t *testing.T) {
	template, err := parse.ReadString(templateString)
	if err != nil {
		t.Error(err)
	}

	testCases := []struct {
		path []interface{}
		keys []string
	}{
		{
			[]interface{}{},
			[]string{
				"Parameters",
				"Resources",
				"Outputs",
			},
		},
		{
			[]interface{}{"Resources"},
			[]string{
				"ZBucket",
				"ARole",
				"ABucket",
			},
		},
	}

	p := newEncoder(New(Options{}), value{template, nil})

	for _, testCase := range testCases {
		p.path = testCase.path
		p.get()

		actual := p.sortKeys()

		if !reflect.DeepEqual(actual, testCase.keys) {
			t.Errorf("Failed sorting.\nExpected: %s\nReceived: %s", testCase.keys, actual)
		}
	}
}
