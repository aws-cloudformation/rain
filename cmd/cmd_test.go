package cmd_test

import (
	"os"

	"github.com/aws-cloudformation/rain/cmd"
	"github.com/aws-cloudformation/rain/console"
)

func init() {
	console.IsTTY = false
}

func Example_tree() {
	os.Args = []string{
		os.Args[0],
		"tree",
		"../examples/success.template",
	}

	cmd.Execute()
	// Output:
	// Resources:
	//   Bucket1:
	//     DependsOn:
	//       Parameters:
	//         - BucketName
}

func Example_diff() {
	os.Args = []string{
		os.Args[0],
		"diff",
		"../examples/success.template",
		"../examples/failure.template",
	}

	cmd.Execute()
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
