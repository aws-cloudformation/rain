package forecast

import (
	"strings"
	"testing"

	"github.com/aws-cloudformation/rain/cft"
	fc "github.com/aws-cloudformation/rain/plugins/forecast"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"gopkg.in/yaml.v3"
)

func TestPredictResourceArn(t *testing.T) {
	input := fc.PredictionInput{
		Source:      &cft.Template{},
		StackName:   "mystack",
		Resource:    &yaml.Node{},
		LogicalId:   "myresource",
		StackExists: false,
		Stack:       types.Stack{},
		TypeName:    "AWS::S3::Bucket",
		Dc:          nil,
		Env:         fc.Env{Partition: "aws", Region: "us-east-1", Account: "123456789012"},
		RoleArn:     "arn:aws:iam::123456789012:role/Admin",
	}

	s3Arn := predictResourceArn(input)
	if !strings.HasPrefix(s3Arn, "arn:aws:s3:::") {
		t.Errorf("Expected arn:aws:s3:::*, got %s", s3Arn)
	}

	input.TypeName = "AWS::EC2::Instance"

	ec2Arn := predictResourceArn(input)
	if !strings.HasPrefix(ec2Arn, "arn:aws:ec2:us-east-1:123456789012:instance/") {
		t.Errorf("Expected arn:aws:ec2:us-east-1:123456789012:instance/*, got %s", ec2Arn)
	}
}
