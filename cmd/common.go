package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws-cloudformation/rain/cfn/diff"
	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/aws-cloudformation/rain/client"
	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/client/s3"
	"github.com/aws-cloudformation/rain/client/sts"
	"github.com/aws-cloudformation/rain/config"
	"github.com/aws-cloudformation/rain/console"
	"github.com/aws-cloudformation/rain/console/run"
	"github.com/aws-cloudformation/rain/console/spinner"
	"github.com/aws-cloudformation/rain/console/text"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

func colouriseStatus(status string) text.Text {
	switch {
	case strings.HasSuffix(status, "_FAILED"):
		return text.Red(status)
	case strings.Contains(status, "ROLLBACK"):
		return text.Orange(status)
	case strings.HasSuffix(status, "_IN_PROGRESS"):
		return text.Orange(status)
	case strings.HasSuffix(status, "_COMPLETE"):
		return text.Green(status)
	default:
		return text.Plain(status)
	}
}

func getStackOutput(stack cloudformation.Stack, onlyChanging bool) string {
	resources, _ := cfn.GetStackResources(*stack.StackName)
	// We ignore errors because it just means we'll list no resources

	out := strings.Builder{}

	out.WriteString(fmt.Sprintf("%s:  # %s\n", *stack.StackName, colouriseStatus(string(stack.StackStatus))))
	if stack.StackStatusReason != nil {
		out.WriteString(fmt.Sprintf("  Message: %s\n", text.Yellow(*stack.StackStatusReason)))
	}

	if len(stack.Parameters) > 0 {
		out.WriteString("  Parameters:\n")
		for _, param := range stack.Parameters {
			out.WriteString(fmt.Sprintf("    %s: %s\n", *param.ParameterKey, text.Yellow(*param.ParameterValue)))
		}
	}

	if len(stack.Outputs) > 0 {
		out.WriteString("  Outputs:\n")
		for _, output := range stack.Outputs {
			out.WriteString(fmt.Sprintf("    %s: %s\n", *output.OutputKey, text.Yellow(*output.OutputValue)))
		}
	}

	if len(resources) > 0 {
		out.WriteString("  Resources:\n")
		if !onlyChanging {
			for _, resource := range resources {
				// Long output
				out.WriteString(fmt.Sprintf("    %s:  # %s\n", *resource.LogicalResourceId, colouriseStatus(string(resource.ResourceStatus))))
				out.WriteString(fmt.Sprintf("      Type: %s\n", text.Yellow(*resource.ResourceType)))
				if resource.PhysicalResourceId != nil {
					out.WriteString(fmt.Sprintf("      PhysicalID: %s\n", text.Yellow(*resource.PhysicalResourceId)))
				}
				if resource.ResourceStatusReason != nil {
					out.WriteString(fmt.Sprintf("      Message: %s\n", text.Yellow(*resource.ResourceStatusReason)))
				}
			}
		} else {
			completed := make([]string, 0)
			updating := strings.Builder{}

			for _, resource := range resources {
				if resourceHasSettled(resource) {
					completed = append(completed, *resource.LogicalResourceId)
				} else {
					// Short output
					updating.WriteString(fmt.Sprintf("      %s: %s  # %s\n",
						*resource.LogicalResourceId,
						text.Yellow(*resource.ResourceType),
						colouriseStatus(string(resource.ResourceStatus)),
					))
				}
			}

			if len(completed) > 0 {
				out.WriteString(fmt.Sprintf("    Complete: %s\n",
					text.Green(strings.Join(completed, " ")),
				))
			}

			if len(updating.String()) > 0 {
				out.WriteString(fmt.Sprintf("    Changing:\n%s\n",
					updating.String(),
				))
			}
		}
	}

	return out.String()
}

func getRainBucket() string {
	accountId, err := sts.GetAccountId()
	if err != nil {
		panic(fmt.Errorf("Unable to get account ID: %s", err))
	}

	bucketName := fmt.Sprintf("rain-artifacts-%s-%s", accountId, client.Config().Region)

	config.Debugf("Artifact bucket: %s", bucketName)

	if !s3.BucketExists(bucketName) {
		err := s3.CreateBucket(bucketName)
		if err != nil {
			panic(fmt.Errorf("Unable to create artifact bucket '%s': %s", bucketName, err))
		}
	}

	return bucketName
}

func colouriseDiff(d diff.Diff, longFormat bool) string {
	output := strings.Builder{}

	for _, line := range strings.Split(format.Diff(d, format.Options{Compact: !longFormat}), "\n") {
		switch {
		case strings.HasPrefix(line, diff.Added.String()):
			output.WriteString(text.Green(line).String())
		case strings.HasPrefix(line, diff.Removed.String()):
			output.WriteString(text.Red(line).String())
		case strings.HasPrefix(line, diff.Changed.String()):
			output.WriteString(text.Orange(line).String())
		case strings.HasPrefix(line, diff.Involved.String()):
			output.WriteString(text.Grey(line).String())
		default:
			output.WriteString(line)
		}

		output.WriteString("\n")
	}

	return output.String()
}

func waitForStackToSettle(stackName string) string {
	// Start the timer
	spinner.Timer()

	stackId := stackName

	for {
		stack, err := cfn.GetStack(stackId)
		if err != nil {
			panic(fmt.Errorf("Operation failed: %s", err))
		}

		// Refresh the stack ID so we can deal with deleted stacks ok
		stackId = *stack.StackId

		output := getStackOutput(stack, true)

		// Send the output first
		if console.IsTTY {
			console.Clear(output)
			spinner.Update()
		}

		// Figure out how many are complete
		updating := 0
		resources, _ := cfn.GetStackResources(*stack.StackName)
		for _, resource := range resources {
			if !resourceHasSettled(resource) {
				updating++
			}
		}
		if updating > 0 {
			rs := "resources"
			if updating == 1 {
				rs = "resource"
			}

			spinner.Status(fmt.Sprintf("(%d %s remaining)", updating, rs))
		}

		// Check to see if we've finished
		if stackHasSettled(stack) {
			spinner.Stop()
			console.Clear(output)
			return string(stack.StackStatus)
		}

		time.Sleep(time.Second * 2)
	}
}

func statusIsSettled(status string) bool {
	if strings.HasSuffix(status, "_COMPLETE") || strings.HasSuffix(status, "_FAILED") {
		return true
	}
	return false
}

func stackHasSettled(stack cloudformation.Stack) bool {
	return statusIsSettled(string(stack.StackStatus))
}

func resourceHasSettled(resource cloudformation.StackResource) bool {
	return statusIsSettled(string(resource.ResourceStatus))
}

func runAws(args ...string) (string, error) {
	if config.Profile != "" {
		args = append(args, "--profile", config.Profile)
	}

	if config.Region != "" {
		args = append(args, "--region", config.Region)
	}

	return run.Capture("aws", args...)
}
