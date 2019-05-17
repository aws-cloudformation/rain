package cmd

import (
	"errors"
	"fmt"

	"github.com/aws-cloudformation/rain/client/cfn"
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
		util.Die(errors.New("Usage: rain cat <stack name>"))
	}

	fmt.Println(cfn.GetStackTemplate(args[0]))
}
