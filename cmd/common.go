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
	switch {
	case strings.HasSuffix(status, "_COMPLETE"):
		return util.Green(status)
	case strings.HasSuffix(status, "_IN_PROGRESS"):
		return util.Orange(status)
	case strings.HasSuffix(status, "_FAILED"):
		return util.Red(status)
	default:
		return util.Plain(status)
	}
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
		out.WriteString(fmt.Sprintf("  Message: %s\n", util.Yellow(*stack.StackStatusReason)))
	}

	if len(stack.Parameters) > 0 {
		out.WriteString("  Parameters:\n")
		for _, param := range stack.Parameters {
			out.WriteString(fmt.Sprintf("    %s: %s\n", *param.ParameterKey, util.Yellow(*param.ParameterValue)))
		}
	}

	if len(stack.Outputs) > 0 {
		out.WriteString("  Outputs:\n")
		for _, output := range stack.Outputs {
			out.WriteString(fmt.Sprintf("    %s: %s\n", *output.OutputKey, util.Yellow(*output.OutputValue)))
		}
	}

	if len(resources) > 0 {
		out.WriteString("  Resources:\n")
		for _, resource := range resources {
			out.WriteString(fmt.Sprintf("    %s:  # %s\n", *resource.LogicalResourceId, colouriseStatus(string(resource.ResourceStatus))))
			out.WriteString(fmt.Sprintf("      Type: %s\n", util.Yellow(*resource.ResourceType)))
			if resource.PhysicalResourceId != nil {
				out.WriteString(fmt.Sprintf("      PhysicalID: %s\n", util.Yellow(*resource.PhysicalResourceId)))
			}
			if resource.ResourceStatusReason != nil {
				out.WriteString(fmt.Sprintf("      Message: %s\n", util.Yellow(*resource.ResourceStatusReason)))
			}
		}
	}

	return out.String()
}

func getRainBucket() string {
	accountId, err := sts.GetAccountId()
	if err != nil {
		util.Die(err)
	}

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
		switch {
		case strings.HasPrefix(line, ">>> "):
			output.WriteString(util.Green(line).String())
		case strings.HasPrefix(line, "<<< "):
			output.WriteString(util.Red(line).String())
		case strings.HasPrefix(line, "||| "):
			output.WriteString(util.Orange(line).String())
		default:
			output.WriteString(line)
		}

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

	util.ClearScreen()

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

	lastOutput := ""

	for {
		select {
		case output := <-outputs:
			if util.IsTTY {
				util.ClearScreen()
				fmt.Println(output)
			}

			lastOutput = output
		case status := <-finished:
			// Display the final status
			util.ClearScreen()
			fmt.Println(lastOutput)

			return status
		default:
			// Allow the display to update regardless
		}

		// Display timer
		if util.IsTTY {
			util.ClearLine()
			fmt.Print(spin[count])
			fmt.Print(" ")
			fmt.Print(time.Now().Sub(start).Truncate(time.Second))

			count = (count + 1) % len(spin)

			time.Sleep(time.Second / 2)
		}
	}
}
