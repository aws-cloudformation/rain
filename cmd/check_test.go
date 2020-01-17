package cmd_test

import (
	"os"

	"github.com/aws-cloudformation/rain/cmd"
)

func Example_check_help() {
	os.Args = []string{
		os.Args[0],
		"help",
		"check",
	}

	cmd.Execute()
	// Output:
	// Reads the specified CloudFormation template and validates it against the current CloudFormation specification.
	//
	// Usage:
	//   rain check <template file>
	//
	// Flags:
	//   -h, --help   help for check
	//
	// Global Flags:
	//       --debug            Output debugging information
	//   -p, --profile string   AWS profile name; read from the AWS CLI configuration file
	//   -r, --region string    AWS region to use
}
