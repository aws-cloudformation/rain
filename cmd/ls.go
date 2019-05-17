package cmd

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/util"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

func init() {
	Commands["ls"] = Command{
		Type: STACK,
		Run:  lsCommand,
		Help: "List running stacks",
	}
}

func colouriseStatus(status string) util.Text {
	colour := util.None

	switch {
	case status == "DELETE_COMPLETE":
		colour = util.Grey
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

	fmt.Println(table.String())
}

func outputStack(stack cloudformation.Stack, fullscreen bool) {
	resources, err := cfn.GetStackResources(*stack.StackName)
	if err != nil {
		util.Die(err)
	}

	if fullscreen {
		fmt.Print("\033[0;0H\033[2J")
	}

	fmt.Printf("%s: %s\n", *stack.StackName, colouriseStatus(string(stack.StackStatus)))
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

func listStack(name string) {
	stack, err := cfn.GetStack(name)
	if err != nil {
		util.Die(err)
	}

	outputStack(stack, false)
}

func lsCommand(args []string) {
	if len(args) == 0 {
		listStacks()
	} else if len(args) == 1 {
		listStack(args[0])
	}
}
