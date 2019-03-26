package cmd

import (
	"os"
	"os/exec"
)

func init() {
	Commands["ls"] = Command{
		Func: lsCommand,
		Help: "List running CloudFormation stacks",
	}
}

func lsCommand(args []string) {
	cmd := exec.Command("aws", "cloudformation", "list-stacks", "--query", "StackSummaries[].[StackName,StackStatus]", "--output", "table")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}
