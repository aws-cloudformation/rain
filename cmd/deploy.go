package cmd

import (
	"fmt"
	"os"
	"path/filepath"
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
	ch := make(chan bool)
	defer close(ch)

	go func() {
		cfn.WaitUntilStackCreateComplete(stackName)
		ch <- true
	}()

	for {
		select {
		case <-ch:
			listStack(stackName, true)
			fmt.Println()
			fmt.Println("Successfully deployed " + stackName)
			return
		default:
			listStack(stackName, true)
		}
		time.Sleep(2 * time.Second)
	}
}
