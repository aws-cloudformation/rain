package cmd_test

import (
	"os"

	"github.com/aws-cloudformation/rain/cmd"
)

func Example_ls_help() {
	os.Args = []string{
		os.Args[0],
		"help",
		"ls",
	}

	cmd.Execute(cmd.Rain)
	// Output:
	// Displays a list of all running stacks or the contents of <stack> if provided.
	//
	// Usage:
	//   rain ls <stack>
	//
	// Aliases:
	//   ls, list
	//
	// Flags:
	//   -a, --all      List stacks across all regions
	//   -h, --help     help for ls
	//   -n, --nested   Show nested stacks (hidden by default)
	//
	// Global Flags:
	//       --debug            Output debugging information
	//   -p, --profile string   AWS profile name; read from the AWS CLI configuration file
	//   -r, --region string    AWS region to use
}
