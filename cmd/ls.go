package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/util"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

var liveStatuses = []cloudformation.StackStatus{
	"CREATE_IN_PROGRESS",
	"CREATE_FAILED",
	"CREATE_COMPLETE",
	"ROLLBACK_IN_PROGRESS",
	"ROLLBACK_FAILED",
	"ROLLBACK_COMPLETE",
	"DELETE_IN_PROGRESS",
	"DELETE_FAILED",
	"UPDATE_IN_PROGRESS",
	"UPDATE_COMPLETE_CLEANUP_IN_PROGRESS",
	"UPDATE_COMPLETE",
	"UPDATE_ROLLBACK_IN_PROGRESS",
	"UPDATE_ROLLBACK_FAILED",
	"UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS",
	"UPDATE_ROLLBACK_COMPLETE",
	"REVIEW_IN_PROGRESS",
}

func init() {
	Commands["ls"] = Command{
		Type: STACK,
		Run:  lsCommand,
		Help: "List running stacks",
	}
}

func listStacks() {
	req := cfnClient.ListStacksRequest(&cloudformation.ListStacksInput{
		StackStatusFilter: liveStatuses,
	})

	table := util.NewTable("Name", "Status")

	p := req.Paginate()

	for p.Next() {
		page := p.CurrentPage()

		for _, s := range page.StackSummaries {
			table.Append(*s.StackName, s.StackStatus)
		}
	}

	if err := p.Err(); err != nil {
		panic(err)
	}

	fmt.Println(table.String())
}

func listStack(name string) {
	// Get the stack properties
	stackReq := cfnClient.DescribeStacksRequest(&cloudformation.DescribeStacksInput{
		StackName: &name,
	})

	stackRes, err := stackReq.Send()
	if err != nil {
		panic(err)
	}

	// Get the resources
	resourceReq := cfnClient.DescribeStackResourcesRequest(&cloudformation.DescribeStackResourcesInput{
		StackName: &name,
	})

	resourceRes, err := resourceReq.Send()
	if err != nil {
		panic(err)
	}

	// Now print it!

	fmt.Printf("%s:\n", name)
	fmt.Println()

	fmt.Printf("  Status: %s\n", stackRes.Stacks[0].StackStatus)
	fmt.Println()

	fmt.Println("  Parameters:")
	for _, param := range stackRes.Stacks[0].Parameters {
		fmt.Printf("    %s: %s\n", *param.ParameterKey, *param.ParameterValue)
	}
	fmt.Println()

	fmt.Println("  Outputs:")
	for _, output := range stackRes.Stacks[0].Outputs {
		fmt.Printf("    %s: %s\n", *output.OutputKey, *output.OutputValue)
	}
	fmt.Println()

	fmt.Println("  Resources:")
	for _, resource := range resourceRes.StackResources {
		fmt.Printf("    %s: %s (%s)\n", *resource.LogicalResourceId, *resource.PhysicalResourceId, *resource.ResourceType)
	}
	fmt.Println()
}

func lsCommand(args []string) {
	if len(args) == 0 {
		listStacks()
	} else if len(args) == 1 {
		listStack(args[0])
	}
}
