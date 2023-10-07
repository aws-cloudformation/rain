package logs

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/ui"

	"github.com/spf13/cobra"
)

var allLogs = false
var chart = false

// Cmd is the logs command's entrypoint
var Cmd = &cobra.Command{
	Use:   "logs <stack> (<resource>)",
	Short: "Show the event log for the named stack",
	Long: `Shows the event log for a stack and its nested stack. Optionally, filter by a specific resource by name, or see a gantt chart of the most recent stack action.

By default, only show log entries that contain a useful message (e.g. a failure message).
You can use the "--all" / "-a" flag to change this behaviour.`,
	Args:                  cobra.RangeArgs(1, 2),
	Aliases:               []string{"log"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]
		resourceName := ""
		if len(args) > 1 {
			resourceName = args[1]
		}

		if !chart {
			// Get logs
			logs, err := getLogs(stackName, resourceName)
			if err != nil {
				panic(ui.Errorf(err, "failed to get logs for stack '%s'", stackName))
			}

			if len(logs) == 0 {
				if allLogs {
					fmt.Println("No interesting log messages to display.")
				} else {
					fmt.Println("No interesting log messages to display. To see everything, use the --all flag")
				}
			} else {
				printLogs(logs)
			}
		} else {
			err := createChart(stackName)
			if err != nil {
				panic(ui.Errorf(err, "failed to generate chart for stack '%s'", stackName))
			}
		}
	},
}

func init() {
	Cmd.Flags().BoolVarP(&allLogs, "all", "a", false, "include uninteresting logs")
	Cmd.Flags().BoolVarP(&chart, "chart", "c", false, "Output a gantt chart of the most recent action as an html file")
	Cmd.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
}
