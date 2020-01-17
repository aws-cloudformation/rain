package cmd_test

import (
	"os"

	"github.com/aws-cloudformation/rain/cmd"
)

func Example_logs_help() {
	os.Args = []string{
		os.Args[0],
		"help",
		"logs",
	}

	cmd.Execute()
	// Output:
	// Shows a nicely-formatted list of the event log for the named stack, optionally limiting the results to a single resource.
	//
	// By default, rain will only show log entries that contain a message, for example a failure reason. You can use flags to change this behaviour.
	//
	// Usage:
	//   rain logs <stack> (<resource>)
	//
	// Aliases:
	//   logs, log
	//
	// Flags:
	//   -a, --all    Include uninteresting logs
	//   -h, --help   help for logs
	//   -l, --long   Display full details
	//   -t, --time   Show results in order of time instead of grouped by resource
	//
	// Global Flags:
	//       --debug            Output debugging information
	//   -p, --profile string   AWS profile name; read from the AWS CLI configuration file
	//   -r, --region string    AWS region to use
}
