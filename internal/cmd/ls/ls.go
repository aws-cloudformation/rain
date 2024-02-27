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
				if len(args) != 2 {
					panic("Usage: rain ls -c stackName changeSetName")
				}
				stackName := args[0]
				changeSetName := args[1]
				spinner.Push("Fetching changeset details")
				cs, err := cfn.GetChangeSet(stackName, changeSetName)
				if err != nil {
					panic(ui.Errorf(err, "failed to get changeset '%s'", changeSetName))
				}
				out := ""
				out += fmt.Sprintf("Arn: %v\n", *cs.ChangeSetId)
				out += fmt.Sprintf("Created: %v\n", cs.CreationTime)
				descr := ""
				if cs.Description != nil {
					descr = *cs.Description
				}
				out += fmt.Sprintf("Description: %v\n", descr)
				reason := ""
				if cs.StatusReason != nil {
					reason = "(" + *cs.StatusReason + ")"
				}
				out += fmt.Sprintf("Status: %v/%v %v\n",
					ui.ColouriseStatus(string(cs.ExecutionStatus)),
					ui.ColouriseStatus(string(cs.Status)),
					reason)
				out += "Parameters: "
				if len(cs.Parameters) == 0 {
					out += "(None)\n"
				} else {
					out += "\n"
				}
				for _, p := range cs.Parameters {
					k, v := "", ""
					if p.ParameterKey != nil {
						k = *p.ParameterKey
					}
					if p.ParameterValue != nil {
						v = *p.ParameterValue
					}
					out += fmt.Sprintf("    %s: %s", k, v)
				}
				// TODO: Convert changes to table
				out += "Changes: \n"
				for _, csch := range cs.Changes {
					if csch.ResourceChange == nil {
						continue
					}
					change := csch.ResourceChange
					rid := ""
					if change.LogicalResourceId != nil {
						rid = *change.LogicalResourceId
					}
					rt := ""
					if change.ResourceType != nil {
						rt = *change.ResourceType
					}
					pid := ""
					if change.PhysicalResourceId != nil {
						pid = *change.PhysicalResourceId
					}
					replace := ""
					switch string(change.Replacement) {
					case "True":
						replace = " [Replace]"
					case "Conditional":
						replace = " [Might replace]"
					}
					out += fmt.Sprintf("%s%s: %s (%s) %s\n",
						string(change.Action),
						replace,
						rid,
						rt,
						pid)

				}
				// TODO: Paging

				spinner.Pop()

				fmt.Println(out)

			} else {
				stackName := args[0]
				spinner.Push("Fetching stack status")
				stack, err := cfn.GetStack(stackName)
				if err != nil {
					panic(ui.Errorf(err, "failed to list stack '%s'", stackName))
				}

				output := cfn.GetStackSummary(stack, all)
				spinner.Pop()

				fmt.Println(output)
			}
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

				// For changesets, we need to now call ListChangeSets for
				// each stack and see if it has any active changesets
				if changeset {
					out := ""
					for _, stack := range stacks {
						if stack.StackName == nil {
							continue
						}
						config.Debugf("Checking stack %s", *stack.StackName)

						sets, err := cfn.ListChangeSets(*stack.StackName)
						if err != nil {
							panic(err)
						}

						if len(sets) == 0 {
							continue
						}

						out += fmt.Sprintf("Stack: %s\n", *stack.StackName)

						for _, cs := range sets {
							if cs.ChangeSetName == nil {
								continue
							}
							out += fmt.Sprintf("    %s\n", *cs.ChangeSetName)
						}

					}

					fmt.Println(out)

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
