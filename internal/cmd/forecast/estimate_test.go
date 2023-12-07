package forecast

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft/parse"
)

func TestResourceEstimate(t *testing.T) {
	resourceName := "AWS::ACMPCA::Certificate"
	action := Create
	est, err := GetResourceEstimate(resourceName, action)
	if err != nil {
		t.Error(err)
		return
	}
	if est != 3 {
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

  # 30s
  A:
    Type: AWS::S3::Bucket
    DependsOn: B
    Properties:
      BucketName: !Ref N

  # 12
  B: 
    Type: AWS::S3::BucketPolicy
    DependsOn: E

  # 16 
  C:
    Type: AWS::EC2::Instance
    DependsOn: [B, D, F, G]

  # 6s 
  D:
    Type: AWS::EC2::LaunchTemplate
    DependsOn: E

  # 30s 
  E: 
    Type: AWS::S3::Bucket

  # 30s
  F:
    Type: AWS::S3::Bucket

  # 30s
  G:
    Type: AWS::S3::Bucket

`
	/*
			       A   C
				    \ / \ \ \
					 B   D F G
					  \ /
					   E

		    Longest is C-D-E = 72
	*/
	// Parse the template
	tt, err := parse.String(string(template))
	if err != nil {
		t.Error(err)
		return
	}
	// config.Debug = true
	total := PredictTotalEstimate(tt, false)
	expected := 72 // will need to adjust this when we modify the database of estimates
	if total != expected {
		t.Errorf("expected total to be %v, got %v", expected, total)
	}

}
