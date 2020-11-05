package tree_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/tree"
	"github.com/aws-cloudformation/rain/internal/console"
)

func Example_tree() {
	os.Args = []string{
		os.Args[0],
		"../../../test/templates/success.template",
	}

	console.NoColour = true

	tree.Cmd.Execute()
	// Output:
	// Resources:
	//   Bucket1:
	//     DependsOn:
	//       Parameters:
	//         - BucketName
}
