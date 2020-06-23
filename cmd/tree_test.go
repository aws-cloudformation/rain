package cmd_test

import (
	"os"

	"github.com/aws-cloudformation/rain/cmd"
)

func Example_tree() {
	os.Args = []string{
		os.Args[0],
		"tree",
		"../examples/success.template",
	}

	cmd.Execute(cmd.Rain)
	// Output:
	// Resources:
	//   Bucket1:
	//     DependsOn:
	//       Parameters:
	//         - BucketName
}
