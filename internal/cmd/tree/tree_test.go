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

func Example_sub() {

	os.Args = []string{
		os.Args[0],
		"../../../test/templates/fix-320.yaml",
	}

	console.NoColour = true

	tree.Cmd.Execute()
	// Output:
	// Resources:
	//   BackupSelection:
	//     DependsOn:
	//       Parameters:
	//         - AWS::AccountId
	//         - AWS::Partition
	//         - AWS::Region
	//         - AWS::StackName
	//         - BackupPlanId
	//       Resources:
	//         - EfsFileSystem
}
