package rm_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/rm"
)

func Example_rm_help() {
	os.Args = []string{
		os.Args[0],
		"--help",
	}

	rm.Cmd.Execute()
	// Output:
	// Deletes the CloudFormation stack named <stack> and waits for the action to complete.
	//
	// Usage:
	//   rm <stack>
	//
	// Aliases:
	//   rm, remove, del, delete
	//
	// Flags:
	//   -d, --detach   Once removal has started, don't wait around for it to finish.
	//   -f, --force    Do not ask; just delete
	//   -h, --help     help for rm
}
