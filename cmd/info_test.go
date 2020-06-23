package cmd_test

import (
	"os"

	"github.com/aws-cloudformation/rain/cmd"
)

func Example_info_help() {
	os.Args = []string{
		os.Args[0],
		"help",
		"info",
	}

	cmd.Execute(cmd.Rain)
	// Output:
	// Display the AWS account and region that you're configured to use.
	//
	// Usage:
	//   rain info
	//
	// Flags:
	//   -c, --creds   Include current AWS credentials
	//   -h, --help    help for info
	//
	// Global Flags:
	//       --debug            Output debugging information
	//   -p, --profile string   AWS profile name; read from the AWS CLI configuration file
	//   -r, --region string    AWS region to use
}
