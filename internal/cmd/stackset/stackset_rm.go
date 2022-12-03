package stackset

import (
	"errors"
	"fmt"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/cobra"
)

// StackSetRmCmd is the ls command's entrypoint
var StackSetRmCmd = &cobra.Command{
	Use:                   "rm <stack set>",
	Short:                 "Delete CloudFormation stack sets in a given region",
	Long:                  "Delete CloudFormation stack sets in a given region",
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
			var et *types.StackSetNotEmptyException
			if errors.As(err, &et) {
				fmt.Println("Change set is not empty")
				if console.Confirm(true, "Do you wish to delete all the stack set instances?") {

					spinner.Push("Deleting all stack set instances")
					err := cfn.DeleteAllChangeSetInstances(stackSetName)
					if err != nil {
						panic(ui.Errorf(err, "error while deleting stack set instances "))
					} else {
						spinner.Push("Deleting stack set..")
						err = cfn.DeleteStackSet(stackSetName)
						spinner.Pop()
						if err != nil {
							panic(ui.Errorf(err, "Could not delete stack set '%s'", stackSetName))
						}
					}
				} else {
					panic(errors.New("user cancelled deployment"))
				}

			}

		} else {
			fmt.Println("Success!")
		}

	},
}

func init() {

}
