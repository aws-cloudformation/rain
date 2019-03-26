package cmd

import (
	"codecommit/builders/cfn-cli/util"
	"fmt"
	"os"
)

func init() {
	Commands["cat"] = Command{
		Func: catCommand,
		Help: "Gets a template from a CloudFormation stack",
	}
}

func catDie() {
	fmt.Fprintln(os.Stderr, "Usage: cfn cat <stack name>")
	os.Exit(1)
}

func catCommand(args []string) {
	if len(args) != 1 {
		catDie()
	}

	util.RunAttached(
		"aws",
		"cloudformation",
		"get-template",
		"--stack-name", args[0],
		"--query", "TemplateBody",
		"--output", "text",
	)
}
