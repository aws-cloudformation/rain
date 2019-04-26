package cmd

import (
	"fmt"
	"os"

	"github.com/aws-cloudformation/rain/util"
)

func init() {
	Commands["cat"] = Command{
		Func: catCommand,
		Help: "Gets a template from a CloudFormation stack",
	}
}

func catCommand(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: cfn cat <stack name>")
		os.Exit(1)
	}

	util.RunAttached(
		"aws",
		"cloudformation",
		"get-template",
		"--query", "TemplateBody",
		"--stack-name", args[0],
	)
}
