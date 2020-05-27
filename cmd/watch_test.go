package cmd_test

import (
	"os"

	"github.com/aws-cloudformation/rain/cmd"
)

func Example_watch_help() {
	os.Args = []string{
		os.Args[0],
		"help",
		"watch",
	}

	cmd.Execute()
	// Output:
	// Repeatedly displays the status of a CloudFormation stack. Useful for watching the progress of a deployment started from outside of Rain.
	//
	// Usage:
	//   rain watch <stack>
	//
	// Flags:
	//   -h, --help   help for watch
	//   -w, --wait   Wait for changes to begin rather than refusing to watch an unchanging stack
	//
	// Global Flags:
	//       --debug            Output debugging information
	//   -p, --profile string   AWS profile name; read from the AWS CLI configuration file
	//   -r, --region string    AWS region to use
}
