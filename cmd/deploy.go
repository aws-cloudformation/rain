package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/util"
)

func init() {
	Commands["deploy"] = Command{
		Type: TEMPLATE,
		Help: "Deploy templates to stacks",
		Run:  deployCommand,
	}
}

func deployCommand(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: rain deploy <template> [stack]\n")
		os.Exit(1)
	}

	fn := args[0]

	stackName := ""

	if len(args) > 1 {
		stackName = args[1]
	} else {
		// Pick a stack name based on the folder name
		dir := filepath.Base(filepath.Dir(fn))

		if dir == "." {
			dir, err := os.Getwd()
			if err != nil {
				util.Die(err)
			}
			dir = filepath.Base(dir)
		}

		stackName = dir
	}

	fmt.Printf("Deploying %s => %s\n", filepath.Base(fn), stackName)

	// Start deployment
	cfn.Deploy(fn, stackName)
	cfn.WaitUntilStackExists(stackName)

	for {
		stack, err := cfn.GetStack(stackName)
		if err != nil {
			util.Die(err)
		}

		outputStack(stack, true)

		message := ""

		status := string(stack.StackStatus)

		switch {
		case status == "CREATE_COMPLETE":
			message = "Successfully deployed " + stackName
		case status == "UPDATE_COMPLETE":
			message = "Successfully updated " + stackName
		case strings.Contains(status, "ROLLBACK") && strings.HasSuffix(status, "_COMPLETE"), strings.HasSuffix(status, "_FAILED"):
			message = "Failed deployment: " + stackName
		}

		if message != "" {
			fmt.Println()
			fmt.Println(message)
			return
		}

		time.Sleep(2 * time.Second)
	}
}
