package cmd

import (
	"fmt"
	"os"

	"github.com/aws-cloudformation/rain/util"
)

func init() {
	Commands["cat"] = Command{
		Type: STACK,
		Help: "Get templates from stacks",
		Run:  catCommand,
	}
}

func catCommand(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: rain cat <stack name>")
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
