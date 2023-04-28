package deploy

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
)

func formatChangeSet(stackName, changeSetName string) string {
	status, err := cfn.GetChangeSet(stackName, changeSetName)
	if err != nil {
		panic(ui.Errorf(err, "error getting changeset '%s' for stack '%s'", changeSetName, stackName))
	}

	out := strings.Builder{}

	out.WriteString(fmt.Sprintf("%s:\n", console.Yellow(fmt.Sprintf("Stack %s", ptr.ToString(status.StackName)))))

	// Non-stack resources
	for _, change := range status.Changes {
		if change.ResourceChange.ChangeSetId != nil {
			// Bunch up nested stacks to the end
			continue
		}

		line := fmt.Sprintf("%s %s",
			*change.ResourceChange.ResourceType,
			*change.ResourceChange.LogicalResourceId,
		)

		switch change.ResourceChange.Action {
		case types.ChangeAction("Add"):
			out.WriteString(console.Green("  + " + line))
		case types.ChangeAction("Modify"):
			out.WriteString(console.Blue("  > " + line))
		case types.ChangeAction("Remove"):
			out.WriteString(console.Red("  - " + line))
		}

		out.WriteString("\n")
	}

	// Nested stacks
	for _, change := range status.Changes {
		if change.ResourceChange.ChangeSetId == nil {
			continue
		}

		child := formatChangeSet("", ptr.ToString(change.ResourceChange.ChangeSetId))
		parts := strings.SplitN(child, "\n", 2)
		header, body := parts[0], parts[1]

		switch change.ResourceChange.Action {
		case types.ChangeAction("Add"):
			out.WriteString(console.Green("  + " + header))
		case types.ChangeAction("Modify"):
			out.WriteString(console.Blue("  > " + header))
		case types.ChangeAction("Remove"):
			out.WriteString(console.Red("  - " + header))
		}
		out.WriteString("\n")

		out.WriteString(ui.Indent("  ", body))
		out.WriteString("\n")
	}

	return strings.TrimSpace(out.String())
}

func PackageTemplate(fn string, yes bool) cft.Template {
	// Call RainBucket for side-effects in case we want to force bucket creation
	s3.RainBucket(yes)

	t, err := pkg.File(fn)
	if err != nil {
		panic(ui.Errorf(err, "error packaging template '%s'", fn))
	}

	return t
}

func CheckStack(stackName string) (types.Stack, bool) {
	// Find out if stack exists already
	// If it does and it's not in a good state, offer to wait/delete
	stack, err := cfn.GetStack(stackName)

	stackExists := false
	if err == nil {
		config.Debugf("Stack exists")
		stackExists = true
	}

	spinner.Pause()

	if stackExists {
		switch {
		case stack.StackStatus == types.StackStatusRollbackComplete,
			stack.StackStatus == types.StackStatusReviewInProgress,
			stack.StackStatus == types.StackStatusCreateFailed:

			message := "Existing stack is empty; deleting it."
			fmt.Println(message)

			err := cfn.DeleteStack(stackName, "")
			if err != nil {
				panic(ui.Errorf(err, "unable to delete stack '%s'", stackName))
			}

			status, _ := cfn.WaitForStackToSettle(stackName)

			if status != "DELETE_COMPLETE" {
				panic(fmt.Errorf("failed to delete stack '%s'", stackName))
			}

			console.ClearLines(console.CountLines(message) + 1)
			fmt.Println("Deleted existing, empty stack.")

			stackExists = false
		case !strings.HasSuffix(string(stack.StackStatus), "_COMPLETE"):
			// Can't update
			panic(fmt.Errorf("stack '%s' could not be updated: %s", stackName, ui.ColouriseStatus(string(stack.StackStatus))))
		}
	}

	spinner.Resume()

	return stack, stackExists
}
