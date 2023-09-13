package forecast

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
)

func TestResourceEstimate(t *testing.T) {
	resourceName := "AWS::ACMPCA::Certificate"
	action := Create
	est, err := GetResourceEstimate(resourceName, action)
	if err != nil {
		t.Error(err)
		return
	}
	if est != 1 {
		t.Errorf("expected AWS::ACMPCA::Certificate create to return 1")
	}
}

func TestDependencyEstimate(t *testing.T) {
	template := `
Parameters:

  N:
    Type: String
    Default: "A"

Resources:

  A:
    Type: AWS::S3::Bucket
    DependsOn: B
    Properties:
      BucketName: !Ref N

  B: 
    Type: AWS::S3::BucketPolicy
    DependsOn: E

  C:
    Type: AWS::EC2::Instance
    DependsOn: [B, D]

  D:
    Type: AWS::EC2::LaunchTemplate
    DependsOn: E

  E: 
    Type: AWS::S3::Bucket

`
	// Parse the template
	tt, err := parse.String(string(template))
	if err != nil {
		t.Error(err)
		return
	}
	config.Debug = true
	total := PredictTotalEstimate(tt, false)
	expected := 52 // will need to adjust this when we modify the database of estimates
	if total != expected {
		t.Errorf("expected total to be %v, got %v", expected, total)
	}

}
