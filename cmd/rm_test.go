package cmd_test

import (
	"os"

	"github.com/aws-cloudformation/rain/cmd"
)

func Example_rm_help() {
	os.Args = []string{
		os.Args[0],
		"help",
		"rm",
	}

	cmd.Execute()
	// Output:
	// Deletes the CloudFormation stack named <stack> and waits for the action to complete.
	//
	// Usage:
	//   rain rm <stack>
	//
	// Aliases:
	//   rm, remove, del, delete
	//
	// Flags:
	//   -d, --detach   Once removal has started, don't wait around for it to finish.
	//   -f, --force    Do not ask; just delete
	//   -h, --help     help for rm
	//
	// Global Flags:
	//       --debug            Output debugging information
	//   -p, --profile string   AWS profile name; read from the AWS CLI configuration file
	//   -r, --region string    AWS region to use
}
