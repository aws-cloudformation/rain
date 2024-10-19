package pkg

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
	"gopkg.in/yaml.v3"
)

func TestConstants(t *testing.T) {
	source := `
Parameters:
  Prefix:
    Type: String

Rain:
  Constants:
    Test1: ezbeard-rain-test-constants
    Test2: !Sub ${Prefix}-${Rain::Test1}-SubTest

Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Rain::Constant Test1
  Bucket2:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Rain::Constant Test2
`
	expect := `
Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: ezbeard-rain-test-constants
  Bucket2:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Sub ${Prefix}-ezbeard-rain-test-constants-SubTest
`

	config.Debug = true

	p, err := parse.String(source)
	if err != nil {
		t.Fatal(err)
	}

	tmpl, err := Template(p, ".", nil)
	if err != nil {
		t.Fatal(err)
	}

	et, err := parse.String(expect)
	if err != nil {
		t.Fatal(err)
	}

	d := diff.New(tmpl, et)
	if d.Mode() != "=" {
		t.Errorf("Output does not match expected: %v", d.Format(true))
	}

}

func TestReplaceConstants(t *testing.T) {
	n := &yaml.Node{Kind: yaml.ScalarNode, Value: "${Rain::Test}"}
	constants := make(map[string]*yaml.Node)
	constants["Test"].Value = "Foo"
	err := replaceConstants(n, constants)
	if err != nil {
		t.Fatal(err)
	}
	if n.Value != "Foo" {
		t.Fatalf("Expected Foo, got %s", n.Value)
	}
}
