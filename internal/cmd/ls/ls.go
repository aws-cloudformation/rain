package ls

import (
	"fmt"
	"sort"

	"github.com/aws-cloudformation/rain/internal/cmd"
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

// Cmd is the ls command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "ls <stack>",
	Short:                 "List running CloudFormation stacks",
	Long:                  "Displays a list of all running stacks or the contents of <stack> if provided.",
	Args:                  cobra.MaximumNArgs(1),
	Aliases:               []string{"list"},
	Annotations:           cmd.StackAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			stackName := args[0]

			spinner.Push("Fetching stack status")
			stack, err := cfn.GetStack(stackName)
			if err != nil {
				panic(ui.Errorf(err, "failed to list stack '%s'", stackName))
			}

			output := ui.GetStackSummary(stack, all)
			spinner.Pop()

			fmt.Println(output)
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
				spinner.Push(fmt.Sprintf("Fetching stacks in %s", region))
				aws.SetRegion(region)
				stacks, err := cfn.ListStacks()
				if err != nil {
					panic(ui.Errorf(err, "failed to list stacks"))
				}
				spinner.Pop()

				if len(stacks) == 0 && all {
					continue
				}

				stackNames := make(sort.StringSlice, 0)
				stackMap := make(map[string]*types.StackSummary)
				for _, stack := range stacks {
					stackNames = append(stackNames, *stack.StackName)
					stackMap[*stack.StackName] = stack
				}
				sort.Strings(stackNames)

				fmt.Println(console.Yellow(fmt.Sprintf("CloudFormation stacks in %s:", region)))
				for _, stackName := range stackNames {
					stack := stackMap[stackName]

					if stack.ParentId == nil {
						fmt.Println(ui.Indent("  ", formatStack(stack, stackMap)))
					}
				}
			}

			aws.SetRegion(origRegion)
		}

		// Reset flags
		all = false
	},
}

func init() {
	Cmd.Flags().BoolVarP(&all, "all", "a", false, "List stacks in all regions or if you specify a stack show more details")
}
