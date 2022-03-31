package watch_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/watch"
)

func Example_watch_help() {
	os.Args = []string{
		os.Args[0],
		"--help",
	}

	watch.Cmd.Execute()
	// Output:
	// Repeatedly displays the status of a CloudFormation stack. Useful for watching the progress of a deployment started from outside of Rain.
	//
	// Usage:
	//   watch <stack>
	//
	// Flags:
	//   -h, --help   help for watch
	//   -w, --wait   wait for changes to begin rather than refusing to watch an unchanging stack
}
