package logs

import (
	"fmt"
	"sort"

	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/ui"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/cobra"
)

var timeOrder = false
var longFormat = false
var allLogs = false

// Cmd is the logs command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "logs <stack> (<resource>)",
	Short:                 "Show the event log for the named stack",
	Long:                  "Shows a nicely-formatted list of the event log for the named stack, optionally limiting the results to a single resource.\n\nBy default, rain will only show log entries that contain a message, for example a failure reason. You can use flags to change this behaviour.",
	Args:                  cobra.RangeArgs(1, 2),
	Aliases:               []string{"log"},
	Annotations:           cmd.StackAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]

		// Get logs
		spinner.Push(fmt.Sprintf("Getting logs for stack '%s'", stackName))
		logs, err := cfn.GetStackEvents(stackName)
		if err != nil {
			panic(ui.Errorf(err, "failed to get events for stack '%s'", stackName))
		}
		spinner.Pop()

		// Filter by resource
		if len(args) > 1 {
			newLogs := make([]*types.StackEvent, 0)

			for _, log := range logs {
				if *log.LogicalResourceId == args[1] {
					newLogs = append(newLogs, log)
				}
			}

			logs = newLogs
		}

		// Filter out uninteresting messages
		newLogs := make([]*types.StackEvent, 0)
		for _, log := range logs {
			if allLogs || (log.ResourceStatusReason != nil && !uninterestingMessages[*log.ResourceStatusReason]) {
				newLogs = append(newLogs, log)
			}
		}
		logs = newLogs

		if len(logs) == 0 {
			fmt.Println("No interesting log messages to display. To see everything, use the --all flag")
			return
		}

		// Reverse order
		for i := len(logs)/2 - 1; i >= 0; i-- {
			j := len(logs) - 1 - i
			logs[i], logs[j] = logs[j], logs[i]
		}

		if timeOrder {
			printLogs(logs)
		} else {
			// Group by resource name
			names := make([]string, 0)
			groups := make(map[string][]*types.StackEvent)
			for _, log := range logs {
				name := *log.LogicalResourceId
				if _, ok := groups[name]; !ok {
					groups[name] = make([]*types.StackEvent, 0)
					names = append(names, name)
				}

				groups[name] = append(groups[name], log)
			}
			sort.Strings(names)

			// Print by group
			for i, name := range names {
				groupLogs := groups[name]
				fmt.Printf("%s:  # %s\n", console.Yellow(name), *groupLogs[0].ResourceType)
				printLogs(groupLogs)

				if i < len(names)-1 {
					fmt.Println()
				}
			}
		}
	},
}

func init() {
	Cmd.Flags().BoolVarP(&timeOrder, "time", "t", false, "Show results in order of time instead of grouped by resource")
	Cmd.Flags().BoolVarP(&longFormat, "long", "l", false, "Display full details")
	Cmd.Flags().BoolVarP(&allLogs, "all", "a", false, "Include uninteresting logs")
}
