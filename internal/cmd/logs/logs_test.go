package logs_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/logs"
)

func Example_logs_help() {
	os.Args = []string{
		os.Args[0],
		"--help",
	}

	logs.Cmd.Execute()
	//Output:
	// Shows the event log for a stack and its nested stack. Optionally, filter by a specific resource by name, or see a gantt chart of the most recent stack action.
	//
	// By default, only show log entries that contain a useful message (e.g. a failure message).
	// You can use the --all flag to change this behaviour.
	//
	// Usage:
	//   logs <stack> (<resource>)
	//
	// Aliases:
	//   logs, log
	//
	// Flags:
	//   -a, --all          include uninteresting logs
	//   -c, --chart        Output a gantt chart of the most recent action as an html file
	//   -d, --days int     Age of the logs to display in days
	//       --debug        Output debugging information
	//   -h, --help         help for logs
	//   -l, --length int   Number of logs to display
}
