package bootstrap_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/bootstrap"
)

func Example_bootstrap_help() {
	os.Args = []string{
		os.Args[0],
		"--help",
	}

	bootstrap.Cmd.Execute()
	// Output:
	// Displays a list of all running stacks or the contents of <stack> if provided.
	//
	// Usage:
	//   ls <stack>
	//
	// Aliases:
	//   ls, list
	//
	// Flags:
	//   -a, --all    list stacks in all regions; if you specify a stack, show more details
	//   -h, --help   help for ls
}
