//+build func_test

package pkg_test

import (
	"fmt"
	"testing"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/google/go-cmp/cmp"
)

const hash = "7e81f4270269cd5111c4926e19de731fb38c6dbf07059d14f4591ce5d8ddd770"
const bucket = "rain-artifacts-1234567890-us-east-1"
const region = "us-east-1"

func compare(t *testing.T, in cft.Template, path string, expected interface{}) {
	out, err := pkg.Template(in)
	if err != nil {
		t.Error(err)
	}

	n := out.GetPath(path)

	var actual interface{}
	err = n.Decode(&actual)
	if err != nil {
		t.Error(err)
	}

	if d := cmp.Diff(expected, actual); d != "" {
		t.Error(d)
	}
}

func TestEmbed(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::Embed": "test.txt",
		},
	})

	compare(t, in, "Test", "This: is a test")
}

func TestInclude(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::Include": "test.txt",
		},
	})

	compare(t, in, "Test", map[string]interface{}{"This": "is a test"})
}

func TestS3Http(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::S3Http": "test.txt",
		},
	})

	compare(t, in, "Test", fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, hash))
}

func TestS3(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::S3": "test.txt",
		},
	})

	compare(t, in, "Test", fmt.Sprintf("s3://%s/%s", bucket, hash))
}

func TestS3Object(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::S3": map[string]interface{}{
				"Path":           "test.txt",
				"BucketProperty": "RainS3Bucket",
				"KeyProperty":    "RainS3Key",
			},
		},
	})

	compare(t, in, "Test", map[string]interface{}{
		"RainS3Bucket": bucket,
		"RainS3Key":    hash,
	})
}

func TestRecursion(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Test": map[string]interface{}{
			"Rain::Include": "recurse.yaml",
		},
	})

	compare(t, in, "Test", map[string]interface{}{"This": "is a test"})
}

func TestServerlessFunction(t *testing.T) {
	in, _ := parse.Map(map[string]interface{}{
		"Resources": map[string]interface{}{
			"MyFunction": map[string]interface{}{
				"Type": "AWS::Serverless::Function",
				"Properties": map[string]interface{}{
					"CodeUri": "test.txt",
				},
			},
		},
	})

	compare(t, in, "Resources/MyFunction/Properties/CodeUri", fmt.Sprintf("s3://%s/%s", bucket, hash))
}
