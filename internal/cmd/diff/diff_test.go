package diff_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/diff"
	"github.com/aws-cloudformation/rain/internal/console"
)

func Example_diff() {
	os.Args = []string{
		os.Args[0],
		"../../../test/templates/success.template",
		"../../../test/templates/failure.template",
	}

	console.NoColour = true

	diff.Cmd.Execute()
	// Output:
	// (>) Description: This template fails
	// (-) Parameters: {...}
	// (|) Resources:
	// (|)   Bucket1:
	// (-)     Properties: {...}
	// (+)   Bucket2:
	// (+)     Properties:
	// (+)       BucketName:
	// (+)         Ref: Bucket1
	// (+)     Type: AWS::S3::Bucket
}
