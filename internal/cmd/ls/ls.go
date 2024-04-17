package ls

import (
	"fmt"
	"sort"

	"github.com/aws-cloudformation/rain/internal/config"
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
var changeset = false

func ShowChangeSetsForStack(stackName string) error {
	sets, err := cfn.ListChangeSets(stackName)
	if err != nil {
		return err
	}

	if len(sets) == 0 {
		return nil
	}

	fmt.Printf("  %s\n", stackName)
	for _, cs := range sets {
		if cs.ChangeSetName == nil {
			continue
		}
		fmt.Printf("    %s %v/%v\n",
			*cs.ChangeSetName,
			ui.ColouriseStatus(string(cs.ExecutionStatus)),
			ui.ColouriseStatus(string(cs.Status)))
	}

	return nil
}

// Cmd is the ls command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "ls <stack> [changeset]",
	Short:                 "List running CloudFormation stacks or changesets",
	Long:                  "Displays a list of all running stacks or the contents of <stack> if provided. If the -c arg is supplied, operates on changesets instead of stacks",
	Args:                  cobra.MaximumNArgs(2),
	Aliases:               []string{"list"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {

			if changeset {
				// Get the status of a single changeset

				if len(args) != 2 {
					panic("Usage: rain ls -c stackName changeSetName")
				}

				stackName := args[0]
				changeSetName := args[1]
				showChangeset(stackName, changeSetName)

				return
			}

			// Get the status for a single stack
			stackName := args[0]
			spinner.Push("Fetching stack status")
			stack, err := cfn.GetStack(stackName)
			if err != nil {
				panic(ui.Errorf(err, "failed to list stack '%s'", stackName))
			}

			output := cfn.GetStackSummary(stack, all)
			spinner.Pop()

			fmt.Println(output)
			fmt.Println(console.Yellow("  ChangeSets:"))
			err = ShowChangeSetsForStack(*stack.StackName)
			if err != nil {
				panic(err)
			}
		} else {
			// List all stacks or changesets

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

				// For changesets, we need to now call ListChangeSets for
				// each stack and see if it has any active changesets
				if changeset {
					fmt.Println(console.Yellow(fmt.Sprintf("Stacks with changesets in %s:", region)))
					for _, stack := range stacks {
						if stack.StackName == nil {
							continue
						}
						config.Debugf("Checking stack %s", *stack.StackName)

						err := ShowChangeSetsForStack(*stack.StackName)
						if err != nil {
							panic(err)
						}

					}

				} else {

					stackMap := make(map[string]types.StackSummary)
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
			}

			aws.SetRegion(origRegion)
		}

		// Reset flags
		all = false
	},
}

func init() {
	Cmd.Flags().BoolVarP(&all, "all", "a", false, "list stacks in all regions; if you specify a stack, show more details")
	Cmd.Flags().BoolVarP(&changeset, "changeset", "c", false, "List changesets instead of stacks")
}
