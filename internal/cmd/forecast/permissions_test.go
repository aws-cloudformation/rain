package forecast

import (
	"strings"
	"testing"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"gopkg.in/yaml.v3"
)

func TestPredictResourceArn(t *testing.T) {
	input := PredictionInput{
		source:      cft.Template{},
		stackName:   "mystack",
		resource:    &yaml.Node{},
		logicalId:   "myresource",
		stackExists: false,
		stack:       types.Stack{},
		typeName:    "AWS::S3::Bucket",
		dc:          nil,
		env:         Env{partition: "aws", region: "us-east-1", account: "123456789012"},
		roleArn:     "arn:aws:iam::123456789012:role/Admin",
	}

	s3Arn := predictResourceArn(input)
	if !strings.HasPrefix(s3Arn, "arn:aws:s3:::") {
		t.Errorf("Expected arn:aws:s3:::*, got %s", s3Arn)
	}

	input.typeName = "AWS::EC2::Instance"

	ec2Arn := predictResourceArn(input)
	if !strings.HasPrefix(ec2Arn, "arn:aws:ec2:us-east-1:123456789012:instance/") {
		t.Errorf("Expected arn:aws:ec2:us-east-1:123456789012:instance/*, got %s", ec2Arn)
	}
}
