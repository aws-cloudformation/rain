package fmt_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/fmt"
)

func Example_fmt_help() {
	os.Args = []string{
		os.Args[0],
		"../../../test/templates/success.template",
	}

	fmt.Cmd.Execute()
	// Output:
	// Description: This template succeeds
	//
	// Parameters:
	//   BucketName:
	//     Type: String
	//
	// Resources:
	//   Bucket1:
	//     Type: AWS::S3::Bucket
	//     Properties:
	//       BucketName: !Ref BucketName
}
