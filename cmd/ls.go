package cmd

import (
	"fmt"

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

func listStacks() {
	table := util.NewTable("Name", "Status")

	cfn.ListStacks(func(s cloudformation.StackSummary) {
		table.Append(*s.StackName, s.StackStatus)
	})

	fmt.Println(table.String())
}

func listStack(name string) {
	stack, err := cfn.GetStack(name)
	if err != nil {
		util.Die(err)
	}

	resources := cfn.GetStackResources(name)

	fmt.Printf("%s:\n", name)
	fmt.Println()

	fmt.Printf("  Status: %s\n", stack.StackStatus)
	fmt.Println()

	fmt.Println("  Parameters:")
	for _, param := range stack.Parameters {
		fmt.Printf("    %s: %s\n", *param.ParameterKey, *param.ParameterValue)
	}
	fmt.Println()

	fmt.Println("  Outputs:")
	for _, output := range stack.Outputs {
		fmt.Printf("    %s: %s\n", *output.OutputKey, *output.OutputValue)
	}
	fmt.Println()

	fmt.Println("  Resources:")
	for _, resource := range resources {
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
