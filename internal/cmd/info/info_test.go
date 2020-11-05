package info_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/info"
)

func Example_info_help() {
	os.Args = []string{
		os.Args[0],
		"--help",
	}

	info.Cmd.Execute()
	// Output:
	// Display the AWS account and region that you're configured to use.
	//
	// Usage:
	//   info
	//
	// Flags:
	//   -c, --creds   Include current AWS credentials
	//   -h, --help    help for info
}
