package cmd

import "codecommit/builders/cfn-cli/util"

func init() {
	Commands["ls"] = Command{
		Func: lsCommand,
		Help: "List running CloudFormation stacks",
	}
}

func lsCommand(args []string) {
	util.RunAttached("aws", "cloudformation", "list-stacks", "--query", "StackSummaries[].[StackName,StackStatus]", "--output", "table")
}
