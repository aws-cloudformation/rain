package ls_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/ls"
)

func Example_ls_help() {
	os.Args = []string{
		os.Args[0],
		"--help",
	}

	ls.Cmd.Execute()
	// Output:
	// Displays a list of all running stacks or the contents of <stack> if provided. If the -c arg is supplied, operates on changesets instead of stacks
	//
	// Usage:
	//   ls <stack> [changeset]
	//
	// Aliases:
	//   ls, list
	//
	// Flags:
	//   -a, --all         list stacks in all regions; if you specify a stack, show more details
	//   -c, --changeset   List changesets instead of stacks
	//   -h, --help        help for ls
}
