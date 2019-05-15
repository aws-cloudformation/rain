package cmd

import (
	"fmt"
	"os"

	"github.com/aws-cloudformation/rain/client/cfn"
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

	fmt.Println(cfn.GetStackTemplate(args[0]))
}
