package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/util"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

const (
	pending = util.Orange
	fail    = util.Red
	success = util.Green
	deleted = util.Grey
)

var statusColours = map[string]util.Colour{
	"CREATE_IN_PROGRESS":                           pending,
	"CREATE_FAILED":                                fail,
	"CREATE_COMPLETE":                              success,
	"ROLLBACK_IN_PROGRESS":                         pending,
	"ROLLBACK_FAILED":                              fail,
	"ROLLBACK_COMPLETE":                            success,
	"DELETE_COMPLETE":                              deleted,
	"DELETE_IN_PROGRESS":                           pending,
	"DELETE_FAILED":                                fail,
	"UPDATE_IN_PROGRESS":                           pending,
	"UPDATE_COMPLETE_CLEANUP_IN_PROGRESS":          pending,
	"UPDATE_COMPLETE":                              success,
	"UPDATE_ROLLBACK_IN_PROGRESS":                  pending,
	"UPDATE_ROLLBACK_FAILED":                       fail,
	"UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS": pending,
	"UPDATE_ROLLBACK_COMPLETE":                     success,
	"REVIEW_IN_PROGRESS":                           pending,
}

func init() {
	Commands["ls"] = Command{
		Type: STACK,
		Run:  lsCommand,
		Help: "Listrunningstacks",
	}
}

func listStacks() {
	table := util.NewTable("Name", "Status")

	cfn.ListStacks(func(s cloudformation.StackSummary) {
		table.Append(*s.StackName, colouriseStatus(string(s.StackStatus)))
	})

	fmt.Println(table.String())
}

func colouriseStatus(status string) util.Text {
	colour, ok := statusColours[status]
	if !ok {
		return util.Text{status, util.None}
	}

	return util.Text{status, colour}
}

func listStack(name string, fullscreen bool) {
	stack, err := cfn.GetStack(name)
	if err != nil {
		util.Die(err)
	}

	resources := cfn.GetStackResources(name)

	if fullscreen {
		fmt.Print("\033[0;0H\033[2J")
	}

	fmt.Printf("%s: %s\n", name, colouriseStatus(string(stack.StackStatus)))
	if stack.StackStatusReason != nil {
		fmt.Printf("  Message: %s\n", *stack.StackStatusReason)
	}

	if len(stack.Parameters) > 0 {
		fmt.Println("  Parameters:")
		for _, param := range stack.Parameters {
			fmt.Printf("    %s: %s\n", *param.ParameterKey, *param.ParameterValue)
		}
	}

	if len(stack.Outputs) > 0 {
		fmt.Println("  Outputs:")
		for _, output := range stack.Outputs {
			fmt.Printf("    %s: %s\n", *output.OutputKey, *output.OutputValue)
		}
	}

	fmt.Println("  Resources:")
	for _, resource := range resources {
		fmt.Printf("    %s: %s\n", *resource.LogicalResourceId, colouriseStatus(string(resource.ResourceStatus)))
		fmt.Printf("      Type: %s\n", *resource.ResourceType)
		if resource.PhysicalResourceId != nil {
			fmt.Printf("      PhysicalID: %s\n", *resource.PhysicalResourceId)
		}
		if resource.ResourceStatusReason != nil {
			fmt.Printf("      Message: %s\n", *resource.ResourceStatusReason)
		}
	}
}

func lsCommand(args []string) {
	if len(args) == 0 {
		listStacks()
	} else if len(args) == 1 {
		listStack(args[0], false)
	}
}
