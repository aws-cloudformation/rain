package format

import (
	"github.com/aws-cloudformation/rain/parse"
)

var templateString = `
Parameters:
  Name:
  Type: String

Resources:
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

var template map[string]interface{}

func init() {
	var err error

	template, err = parse.ReadString(templateString)
	if err != nil {
		panic(err)
	}
}

/*
func TestOrdering(t *testing.T) {
	e := newEncoder(New(Options{}), value{template, nil})
}
*/
