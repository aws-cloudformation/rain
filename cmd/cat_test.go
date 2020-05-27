package cmd_test

import (
	"os"

	"github.com/aws-cloudformation/rain/cmd"
)

func Example_cat_help() {
	os.Args = []string{
		os.Args[0],
		"help",
		"cat",
	}

	cmd.Execute()
	// Output:
	// Downloads the template used to deploy <stack> and prints it to stdout.
	//
	// Usage:
	//   rain cat <stack>
	//
	// Flags:
	//   -h, --help          help for cat
	//   -t, --transformed   Get the template with transformations applied by CloudFormation.
	//   -u, --unformatted   Output the template in its raw form and do not attempt to format it.
	//
	// Global Flags:
	//       --debug            Output debugging information
	//   -p, --profile string   AWS profile name; read from the AWS CLI configuration file
	//   -r, --region string    AWS region to use
}
