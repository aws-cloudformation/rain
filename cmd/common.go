package cmd

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/client/s3"
	"github.com/aws-cloudformation/rain/client/sts"
	"github.com/aws-cloudformation/rain/util"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

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
	fmt.Println("CAMEL")

	fmt.Println(table.String())
}

func outputStack(stack cloudformation.Stack, fullscreen bool) {
	resources, _ := cfn.GetStackResources(*stack.StackName)
	// We ignore errors because it just means we'll list no resources

	if fullscreen {
		fmt.Print("\033[0;0H\033[2J")
	}

	fmt.Printf("%s: # %s\n", *stack.StackName, colouriseStatus(string(stack.StackStatus)))
	if stack.StackStatusReason != nil {
		fmt.Printf("  Message: %s\n", util.Text{*stack.StackStatusReason, util.Yellow})
	}

	if len(stack.Parameters) > 0 {
		fmt.Println("  Parameters:")
		for _, param := range stack.Parameters {
			fmt.Printf("    %s: %s\n", *param.ParameterKey, util.Text{*param.ParameterValue, util.Yellow})
		}
	}

	if len(stack.Outputs) > 0 {
		fmt.Println("  Outputs:")
		for _, output := range stack.Outputs {
			fmt.Printf("    %s: %s\n", *output.OutputKey, util.Text{*output.OutputValue, util.Yellow})
		}
	}

	if len(resources) > 0 {
		fmt.Println("  Resources:")
		for _, resource := range resources {
			fmt.Printf("    %s: # %s\n", *resource.LogicalResourceId, colouriseStatus(string(resource.ResourceStatus)))
			fmt.Printf("      Type: %s\n", util.Text{*resource.ResourceType, util.Yellow})
			if resource.PhysicalResourceId != nil {
				fmt.Printf("      PhysicalID: %s\n", util.Text{*resource.PhysicalResourceId, util.Yellow})
			}
			if resource.ResourceStatusReason != nil {
				fmt.Printf("      Message: %s\n", util.Text{*resource.ResourceStatusReason, util.Yellow})
			}
		}
	}
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
