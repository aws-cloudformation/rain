package cat_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/cat"
)

func Example_cat_help() {
	os.Args = []string{
		os.Args[0],
		"--help",
	}

	cat.Cmd.Execute()
	// Output:
	// Downloads the template used to deploy <stack> and prints it to stdout.
	//
	// Usage:
	//   cat <stack>
	//
	// Flags:
	//   -h, --help          help for cat
	//   -t, --transformed   get the template with transformations applied by CloudFormation
	//   -u, --unformatted   output the template in its raw form; do not attempt to format it
}
