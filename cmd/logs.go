package cmd

import (
	"fmt"
	"sort"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/util"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/spf13/cobra"
)

var timeOrder = false
var longFormat = false

func printLongLogs(logs []cloudformation.StackEvent, includeName bool) {
	for _, log := range logs {
		fmt.Printf("- Timestamp: %s\n", util.Yellow(fmt.Sprint(*log.Timestamp)))
		fmt.Printf("  Status: %s\n", colouriseStatus(string(log.ResourceStatus)))
		if includeName {
			fmt.Printf("  Name: %s\n", util.Yellow(*log.LogicalResourceId))
			fmt.Printf("  Type: %s\n", *log.ResourceType)
		}
		fmt.Printf("  PhysicalID: %s\n", util.Yellow(*log.PhysicalResourceId))
		if log.ResourceStatusReason != nil {
			fmt.Printf("  Message: %s\n", util.White(fmt.Sprintf("%q", *log.ResourceStatusReason)))
		}
	}
}

var logsCmd = &cobra.Command{
	Use:                   "logs <stack> (<resource>)",
	Short:                 "Show the event log for the named stack",
	Long:                  "Shows a nicely-formatted list of the event log for the named stack, optionally limiting the results to a single resource.",
	Args:                  cobra.RangeArgs(1, 2),
	Aliases:               []string{"log"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := args[0]

		// Get logs
		logs, err := cfn.GetStackEvents(stackName)
		if err != nil {
			panic(fmt.Errorf("Failed to get events for '%s': %s", stackName, err))
		}

		// Filter by resource
		if len(args) > 1 {
			newLogs := make([]cloudformation.StackEvent, 0)

			for _, log := range logs {
				if *log.LogicalResourceId == args[1] {
					newLogs = append(newLogs, log)
				}
			}

			logs = newLogs
		}

		// Reverse order
		for i := len(logs)/2 - 1; i >= 0; i-- {
			j := len(logs) - 1 - i
			logs[i], logs[j] = logs[j], logs[i]
		}

		if timeOrder {
			if longFormat {
				printLongLogs(logs, true)
			} else {
				for _, log := range logs {
					if log.ResourceStatusReason == nil {
						continue
					}

					fmt.Printf("- %s: %s  # %s\n",
						*log.LogicalResourceId,
						colouriseStatus(string(log.ResourceStatus)),
						*log.ResourceStatusReason,
					)
				}
			}
		} else {
			names := make([]string, 0)

			// Group by resource name
			groups := make(map[string][]cloudformation.StackEvent)
			for _, log := range logs {
				name := *log.LogicalResourceId
				if _, ok := groups[name]; !ok {
					groups[name] = make([]cloudformation.StackEvent, 0)
					names = append(names, name)
				}

				groups[name] = append(groups[name], log)
			}

			sort.Strings(names)

			// Print by group
			for _, name := range names {
				groupLogs := groups[name]

				fmt.Printf("%s:  # %s\n", name, util.Grey(*groupLogs[0].ResourceType))

				if longFormat {
					printLongLogs(groupLogs, false)
				} else {
					for _, log := range groupLogs {
						if log.ResourceStatusReason == nil {
							continue
						} else {
							fmt.Printf("- %s: %s\n",
								colouriseStatus(string(log.ResourceStatus)),
								util.White(fmt.Sprintf("%q", *log.ResourceStatusReason)),
							)
						}
					}
				}

				fmt.Println()
			}
		}
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&timeOrder, "time", "t", false, "Show results in order of time instead of grouped by resource")
	logsCmd.Flags().BoolVarP(&longFormat, "long", "l", false, "Display full details and include uninteresting logg")
	rootCmd.AddCommand(logsCmd)
}
