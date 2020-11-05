package check_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/check"
)

func Example_check_help() {
	os.Args = []string{
		os.Args[0],
		"--help",
	}

	check.Cmd.Execute()
	// Output:
	// Reads the specified CloudFormation template and validates it against the current CloudFormation specification.
	//
	// Usage:
	//   check <template file>
	//
	// Flags:
	//   -h, --help   help for check
}
