package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/util"
)

func init() {
	Commands["rm"] = Command{
		Type: STACK,
		Run:  rmCommand,
		Help: "Delete a stack",
	}
}

func rmCommand(args []string) {
	if len(args) != 1 {
		util.Die(errors.New("Usage: rm <stack name>"))
	}

	stackName := args[0]

	err := cfn.DeleteStack(stackName)
	if err != nil {
		util.Die(err)
	}

	for {
		stack, err := cfn.GetStack(stackName)
		if err != nil {
			util.Die(err)
		}

		outputStack(stack, true)

		message := ""

		status := string(stack.StackStatus)

		switch {
		case status == "DELETE_COMPLETE":
			message = "Successfully deleted " + stackName
		case strings.HasSuffix(status, "_COMPLETE") || strings.HasSuffix(status, "_FAILED"):
			message = "Failed to delete " + stackName
		}

		if message != "" {
			fmt.Println()
			fmt.Println(message)
			return
		}

		time.Sleep(2 * time.Second)
	}
}
