package cmd_test

import (
	"os"

	"github.com/aws-cloudformation/rain/cmd"
)

func Example_fmt_help() {
	os.Args = []string{
		os.Args[0],
		"help",
		"fmt",
	}

	cmd.Execute()
	// Output:
	// Reads the named template and outputs a nicely formatted copy.
	//
	// Usage:
	//   rain fmt <filename>
	//
	// Aliases:
	//   fmt, format
	//
	// Flags:
	//   -c, --compact   Produce more compact output.
	//   -h, --help      help for fmt
	//   -j, --json      Output the template as JSON (default format: YAML).
	//   -v, --verify    Check if the input is already correctly formatted and exit.
	//                   The exit status will be 0 if so and 1 if not.
	//   -w, --write     Write the output back to the file rather than to stdout.
	//
	// Global Flags:
	//       --debug            Output debugging information
	//   -p, --profile string   AWS profile name; read from the AWS CLI configuration file
	//   -r, --region string    AWS region to use
}
