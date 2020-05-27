package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/client/ec2"
	"github.com/aws-cloudformation/rain/console/spinner"
	"github.com/aws-cloudformation/rain/console/text"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/spf13/cobra"
)

var allRegions = false
var showNested = false

func formatStack(stack cloudformation.StackSummary, stackMap map[string]cloudformation.StackSummary) string {
	out := strings.Builder{}

	out.WriteString(fmt.Sprintf("%s: %s\n",
		*stack.StackName,
		colouriseStatus(string(stack.StackStatus)),
	))

	if showNested {
		for _, otherStack := range stackMap {
			if otherStack.ParentId != nil && *otherStack.ParentId == *stack.StackId {
				out.WriteString(indent("- ", formatStack(otherStack, stackMap)))
				out.WriteString("\n")
			}
		}
	}

	return out.String()
}

var lsCmd = &cobra.Command{
	Use:                   "ls <stack>",
	Short:                 "List running CloudFormation stacks",
	Long:                  "Displays a list of all running stacks or the contents of <stack> if provided.",
	Args:                  cobra.MaximumNArgs(1),
	Aliases:               []string{"list"},
	Annotations:           stackAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 1 {
			stackName := args[0]

			stack, err := cfn.GetStack(stackName)
			if err != nil {
				panic(fmt.Errorf("Failed to list stack '%s': %s", stackName, err))
			}

			fmt.Println(getStackOutput(stack, false))
		} else {
			var err error
			regions := []string{client.Config().Region}

			spinner.Status("Fetching region list")
			if allRegions {
				regions, err = ec2.GetRegions()
				if err != nil {
					panic(fmt.Errorf("Unable to get region list: %s", err))
				}
			}

			for _, region := range regions {
				spinner.Status(fmt.Sprintf("Fetching stacks in %s", region))

				client.SetRegion(region)
				stacks, err := cfn.ListStacks()
				if err != nil {
					panic(fmt.Errorf("Failed to list stacks: %s", err))
				}

				if len(stacks) == 0 && allRegions {
					continue
				}

				stackNames := make(sort.StringSlice, 0)
				stackMap := make(map[string]cloudformation.StackSummary)
				for _, stack := range stacks {
					stackNames = append(stackNames, *stack.StackName)
					stackMap[*stack.StackName] = stack
					fmt.Println(stack)
				}
				sort.Strings(stackNames)

				spinner.Stop()

				fmt.Println(text.Yellow(fmt.Sprintf("CloudFormation stacks in %s:", region)))
				for _, stackName := range stackNames {
					stack := stackMap[stackName]

					if stack.ParentId == nil {
						fmt.Println(indent("  ", formatStack(stack, stackMap)))
					}
				}
				fmt.Println()
			}
		}
	},
}

func init() {
	lsCmd.Flags().BoolVarP(&allRegions, "all", "a", false, "List stacks across all regions")
	lsCmd.Flags().BoolVarP(&showNested, "nested", "n", false, "Show nested stacks (hidden by default)")
	Root.AddCommand(lsCmd)
}
