package format

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/google/go-cmp/cmp"
)

func TestYamlToPkl(t *testing.T) {
	input := `
Parameters:
  Name:
    Type: String
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
        BucketName: 
          Ref: Name
Outputs:
  BucketName:
    Value: !Ref BucketName
    Description: The bucket name
    Export:
      Name: ExportedBucketName
`

	expected := `amends "@cfn/template.pkl"
import "@cfn/cloudformation.pkl" as cfn
import "@cfn/aws/s3/bucket.pkl"

Parameters {
    ["Name"] {
        Type = "String"
    }
}

Resources {
    ["MyBucket"] = new bucket.Bucket {
        Type = "AWS::S3::Bucket"
        BucketName = cfn.Ref("Name")

    }

}

Outputs {
    ["BucketName"] = new cfn.Output {
        Value = cfn.Ref("BucketName")
        Description = The bucket name
        Export = new cfn.Export {
            Name = ExportedBucketName
        }
    }
}
`

	template, err := parse.String(input)
	if err != nil {
		t.Fatal(err)
	}

	actual, err := CftToPkl(template, false, "@cfn")
	if err != nil {
		t.Fatal(err)
	}

	if d := cmp.Diff(expected, actual); d != "" {
		t.Errorf(d)
	}
}

func TestYamlToPklBasic(t *testing.T) {
	input := `
Parameters:
  Name:
    Type: String
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
        BucketName: 
          Ref: Name
`

	expected := `
Parameters {
    ["Name"] {
        Type = "String"
    }
}

Resources {
    ["MyBucket"] {
        Type = "AWS::S3::Bucket"
        Properties {
            ["BucketName"] {
                ["Ref"] = "Name"
            }
        }
    }

}
`
	template, err := parse.String(input)
	if err != nil {
		t.Fatal(err)
	}

	actual, err := CftToPkl(template, true, "@cfn")
	if err != nil {
		t.Fatal(err)
	}

	if d := cmp.Diff(expected, actual); d != "" {
		t.Errorf(d)
	}
}
