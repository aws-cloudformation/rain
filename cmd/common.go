package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/client/s3"
	"github.com/aws-cloudformation/rain/client/sts"
	"github.com/aws-cloudformation/rain/diff"
	"github.com/aws-cloudformation/rain/util"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

var spin = []string{
	`-`, `\`, `|`, `/`,
}

func stackExists(stackName string) bool {
	ch := make(chan bool)

	go func() {
		cfn.ListStacks(func(s cloudformation.StackSummary) {
			if *s.StackName == stackName {
				ch <- true
			}
		})

		// Default
		ch <- false
	}()

	return <-ch
}

func colouriseStatus(status string) util.Text {
	colour := util.None

	switch {
	case strings.HasSuffix(status, "_COMPLETE"):
		colour = util.Green
	case strings.HasSuffix(status, "_IN_PROGRESS"):
		colour = util.Orange
	case strings.HasSuffix(status, "_FAILED"):
		colour = util.Red
	}

	return util.Text{status, colour}
}

func listStacks() {
	table := util.NewTable("Name", "Status")

	cfn.ListStacks(func(s cloudformation.StackSummary) {
		table.Append(*s.StackName, colouriseStatus(string(s.StackStatus)))
	})

	table.Sort()

	fmt.Println(table.String())
}

func getStackOutput(stack cloudformation.Stack) string {
	resources, _ := cfn.GetStackResources(*stack.StackName)
	// We ignore errors because it just means we'll list no resources

	out := strings.Builder{}

	out.WriteString(fmt.Sprintf("%s:  # %s\n", *stack.StackName, colouriseStatus(string(stack.StackStatus))))
	if stack.StackStatusReason != nil {
		out.WriteString(fmt.Sprintf("  Message: %s\n", util.Text{*stack.StackStatusReason, util.Yellow}))
	}

	if len(stack.Parameters) > 0 {
		out.WriteString("  Parameters:\n")
		for _, param := range stack.Parameters {
			out.WriteString(fmt.Sprintf("    %s: %s\n", *param.ParameterKey, util.Text{*param.ParameterValue, util.Yellow}))
		}
	}

	if len(stack.Outputs) > 0 {
		out.WriteString("  Outputs:\n")
		for _, output := range stack.Outputs {
			out.WriteString(fmt.Sprintf("    %s: %s\n", *output.OutputKey, util.Text{*output.OutputValue, util.Yellow}))
		}
	}

	if len(resources) > 0 {
		out.WriteString("  Resources:\n")
		for _, resource := range resources {
			out.WriteString(fmt.Sprintf("    %s:  # %s\n", *resource.LogicalResourceId, colouriseStatus(string(resource.ResourceStatus))))
			out.WriteString(fmt.Sprintf("      Type: %s\n", util.Text{*resource.ResourceType, util.Yellow}))
			if resource.PhysicalResourceId != nil {
				out.WriteString(fmt.Sprintf("      PhysicalID: %s\n", util.Text{*resource.PhysicalResourceId, util.Yellow}))
			}
			if resource.ResourceStatusReason != nil {
				out.WriteString(fmt.Sprintf("      Message: %s\n", util.Text{*resource.ResourceStatusReason, util.Yellow}))
			}
		}
	}

	return out.String()
}

func clearScreen() {
	fmt.Print("\033[0;0H\033[2J")
}

func getRainBucket() string {
	accountId := sts.GetAccountId()

	bucketName := fmt.Sprintf("rain-artifacts-%s", accountId)

	if !s3.BucketExists(bucketName) {
		err := s3.CreateBucket(bucketName)
		if err != nil {
			util.Die(err)
		}
	}

	return bucketName
}

func colouriseDiff(d diff.Diff) string {
	output := strings.Builder{}

	for _, line := range strings.Split(diff.Format(d), "\n") {
		colour := util.None

		switch {
		case strings.HasPrefix(line, ">>> "):
			colour = util.Green
		case strings.HasPrefix(line, "<<< "):
			colour = util.Red
		case strings.HasPrefix(line, "||| "):
			colour = util.Orange
		}

		output.WriteString(util.Text{line, colour}.String())
		output.WriteString("\n")
	}

	return output.String()
}

func waitForStackToSettle(stackName string) string {
	// Start the timer
	start := time.Now()
	count := 0

	// Channel for receiving new stack statuses
	outputs := make(chan string)
	finished := make(chan string)

	defer close(outputs)
	defer close(finished)

	clearScreen()

	go func(stackId string) {
		for {
			stack, err := cfn.GetStack(stackId)
			if err != nil {
				util.Die(err)
			}

			// Refresh the stack ID so we can deal with deleted stacks ok
			stackId = *stack.StackId

			// Send the output first
			outputs <- getStackOutput(stack)

			// Check to see if we've finished
			status := string(stack.StackStatus)
			if strings.HasSuffix(status, "_COMPLETE") || strings.HasSuffix(status, "_FAILED") {
				finished <- status
			}

			time.Sleep(time.Second * 2)
		}
	}(stackName)

	for {
		select {
		case output := <-outputs:
			clearScreen()
			fmt.Println(output)
		default:
			// Allow the display to update regardless
		}

		select {
		case status := <-finished:
			return status
		default:
			// Allow the display to update regardless
		}

		// Display timer
		fmt.Print(spin[count])
		fmt.Print(" ")
		fmt.Print(time.Now().Sub(start).Truncate(time.Second))
		fmt.Print("\033[0G")

		count = (count + 1) % len(spin)

		time.Sleep(time.Second / 2)
	}

	return ""
}
