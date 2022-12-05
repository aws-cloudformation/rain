package stackset

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aws-cloudformation/rain/internal/ui"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/ec2"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/cobra"
)

var all = false

// StackSetLsCmd is the ls command's entrypoint
var StackSetLsCmd = &cobra.Command{
	Use:                   "ls <stack set>",
	Short:                 "List CloudFormation stack sets in a given region",
	Long:                  "List CloudFormation stack sets in a given region",
	Args:                  cobra.MaximumNArgs(1),
	Aliases:               []string{"list"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			stackSetName := args[0]

			spinner.Push("Fetching stack set status")
			stackSet, err := cfn.GetStackSet(stackSetName)
			if err != nil {
				panic(ui.Errorf(err, "failed to list stack set '%s'", stackSetName))
			}
			spinner.Pop()

			output := ui.GetStackSetSummary(stackSet, all)
			fmt.Println(output)

			fmt.Println(ui.Indent("  ", formatStackSetInstances(string(stackSetName))))
		} else {
			var err error
			regions := []string{aws.Config().Region}

			if all {
				spinner.Push("Fetching region list")
				regions, err = ec2.GetRegions()
				if err != nil {
					panic(ui.Errorf(err, "unable to get region list"))
				}
				spinner.Pop()
			}

			origRegion := aws.Config().Region

			for _, region := range regions {
				spinner.Push(fmt.Sprintf("Fetching stack sets in %s", region))
				aws.SetRegion(region)
				stackSets, err := cfn.ListStackSets()
				if err != nil {
					panic(ui.Errorf(err, "failed to list stack sets"))
				}
				spinner.Pop()

				if len(stackSets) == 0 && all {
					continue
				}

				stackSetNames := make(sort.StringSlice, 0)
				stackSetMap := make(map[string]types.StackSetSummary)
				for _, stack := range stackSets {
					if stack.Status != types.StackSetStatusDeleted {
						stackSetNames = append(stackSetNames, *stack.StackSetName)
						stackSetMap[*stack.StackSetName+region] = stack
					}
				}
				sort.Strings(stackSetNames)

				fmt.Println(console.Yellow(fmt.Sprintf("CloudFormation stack sets in %s:", region)))
				for _, stackSetName := range stackSetNames {
					out := strings.Builder{}
					out.WriteString(fmt.Sprintf("%s: %s\n",
						stackSetName,
						ui.ColouriseStatus(string(stackSetMap[stackSetName+region].Status)),
					))
					fmt.Println(ui.Indent("  ", out.String()))
				}
			}

			aws.SetRegion(origRegion)
		}

		// Reset flags
		all = false
	},
}

func init() {
	StackSetLsCmd.Flags().BoolVarP(&all, "all", "a", false, "list stacks in all regions; if you specify a stack, show more details")
}

func formatStackSetInstances(stackSetName string) string {
	out := strings.Builder{}
	out.WriteString(console.Yellow("Instances: (StackSet Name/Account/Region/Status/Reason)\n"))
	spinner.Push(fmt.Sprintf("Fetching stack set instances for '%s'", stackSetName))
	stackSetInstances, err := cfn.ListStackSetInstances(stackSetName)
	if err != nil {
		panic(ui.Errorf(err, "failed to list stack set instancess"))
	}
	spinner.Pop()

	for _, instance := range stackSetInstances {
		out.WriteString(fmt.Sprintf(" - %s / %s / %s / %s ",
			*instance.StackSetId,
			*instance.Account,
			*instance.Region,
			ui.ColouriseStatus(string(instance.StackInstanceStatus.DetailedStatus)),
		))
		if instance.StatusReason != nil {
			out.WriteString(fmt.Sprintf("/ %s \n", *instance.StatusReason))
		} else {
			out.WriteString("\n")
		}

	}
	out.WriteString("\n")

	return out.String()
}
