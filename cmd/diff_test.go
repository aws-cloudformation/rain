package cmd_test

import (
	"os"

	"github.com/aws-cloudformation/rain/cmd"
)

func Example_diff() {
	os.Args = []string{
		os.Args[0],
		"diff",
		"../examples/success.template",
		"../examples/failure.template",
	}

	cmd.Execute(cmd.Rain)
	// Output:
	// (>) Description: This template fails
	// (-) Parameters: {...}
	// (|) Resources:
	// (|)   Bucket1:
	// (-)     Properties: {...}
	// (+)   Bucket2:
	// (+)     Properties:
	// (+)       BucketName: !Ref Bucket1
	// (+)     Type: "AWS::S3::Bucket"
}
