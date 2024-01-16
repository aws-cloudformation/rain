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
	// Downloads the template or the configuration file used to deploy <stack> and prints it to stdout.
	//
	// The  `--config` flag can be used to get the rain config file for the stack instead of the template.
	//
	// Usage:
	//   cat <stack>
	//
	// Flags:
	//   -c, --config        output the config file for the existing stack
	//   -h, --help          help for cat
	//   -t, --transformed   get the template with transformations applied by CloudFormation
	//   -u, --unformatted   output the template in its raw form; do not attempt to format it
}
