package stackset

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/cobra"
)

// StackSetRmCmd is the rm command's entrypoint
var StackSetRmCmd = &cobra.Command{
	Use:                   "rm <stackset>",
	Short:                 "Delete a CloudFormation stack set and/or its instances.",
	Long:                  "Delete a CloudFormation stack set <stackset> and/or its instances.",
	Args:                  cobra.ExactArgs(1),
	Aliases:               []string{"delete", "remove"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackSetName := args[0]
		config.Debugf("Deleting stack set: %s\n", stackSetName)

		stackSet, err := cfn.GetStackSet(stackSetName)
		if err != nil {
			panic(ui.Errorf(err, "Could not find stack set '%s'", stackSetName))
		} else if stackSet.Status == types.StackSetStatusDeleted {
			panic(ui.Errorf(err, "Stack set '%s' is already in DELETED state ", stackSetName))
		}

		spinner.Push("Deleting stack set..")
		err = cfn.DeleteStackSet(stackSetName)
		spinner.Pop()
		if err != nil {
			var notEmptyException *types.StackSetNotEmptyException
			if errors.As(err, &notEmptyException) {

				instancesOut, instances := getStackInstances(stackSetName)
				fmt.Printf("%s", instancesOut)
				inputString := console.Ask("Select instances number to delete or 0 to delete all. For multiple selects separate with comma:")

				accounts, regions, deleteAll := convertInputString(inputString, instances)

				spinner.Push("Deleting stack set instances...")
				if deleteAll {
					err = cfn.DeleteAllStackSetInstances(stackSetName, !detach, false)
				} else {
					err = cfn.DeleteStackSetInstances(stackSetName, accounts, regions, !detach, false)
				}
				spinner.Pop()

				if err != nil {
					panic(ui.Errorf(err, "error while deleting stack set instances "))
				}

				if deleteAll && console.Confirm(true, "Do you want to delete stack set now?") {
					spinner.Push("Deleting stack set...")
					err = cfn.DeleteStackSet(stackSetName)
					spinner.Pop()
					if err != nil {
						panic(ui.Errorf(err, "Could not delete stack set '%s'", stackSetName))
					} else {
						fmt.Println("Stack set deletion has been completed.")
					}
				}
			} else {
				panic(ui.Errorf(err, "Could not delete stack set '%s'", stackSetName))
			}

		} else {
			fmt.Println("Success!")
		}

	},
}

func init() {
	StackSetRmCmd.Flags().BoolVarP(&detach, "detach", "d", false, "once delete has started, don't wait around for it to finish")
}

func getStackInstances(stackSetName string) (string, []types.StackInstanceSummary) {
	out := strings.Builder{}
	out.WriteString(console.Yellow("Instances (StackID/Account/Region/Status/Reason):\n"))
	spinner.Push(fmt.Sprintf("Fetching stack set instances for '%s'", stackSetName))
	instances, err := cfn.ListStackSetInstances(stackSetName)
	if err != nil {
		panic(ui.Errorf(err, "failed to list stack set instancess"))
	}
	spinner.Pop()

	if len(instances) == 0 {
		out.WriteString(" - \n")
		return out.String(), instances
	}

	for i, instance := range instances {
		stackId := (*instance.StackId)[strings.Index(*instance.StackId, "stack/")+6 : len(*instance.StackId)]
		out.WriteString(fmt.Sprintf(" [%d] - %s / %s / %s / %s ",
			i+1,
			stackId,
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

	return out.String(), instances
}

// Converts user input string to set of accounts and regions associated with selected instances
func convertInputString(inputString string, instances []types.StackInstanceSummary) ([]string, []string, bool) {
	accounts := []string{}
	regions := []string{}

	selectNumbers := strings.Split(inputString, ",")
	for _, num := range selectNumbers {
		numInt, err := strconv.Atoi(num)
		if err != nil || numInt < 0 || numInt > len(instances) {
			panic(ui.Errorf(err, "Invalid input: '%s'", num))
		}
		if numInt == 0 {
			return []string{}, []string{}, true
		}
		accounts = append(accounts, *instances[numInt-1].Account)
		regions = append(regions, *instances[numInt-1].Region)
	}
	return accounts, regions, false
}
