package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/diff"
	"github.com/aws-cloudformation/rain/util"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/awslabs/aws-cloudformation-template-formatter/parse"
)

func init() {
	Commands["deploy"] = Command{
		Type: TEMPLATE,
		Help: "Deploy templates to stacks",
		Run:  deployCommand,
	}
}

func deployCommand(args []string) {
	var err error

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: rain deploy <template> [stack]\n")
		os.Exit(1)
	}

	fn := args[0]

	stackName := ""

	if len(args) > 1 {
		stackName = args[1]
	} else {
		dir := filepath.Base(filepath.Dir(fn))

		if dir == "." {
			dir, err = os.Getwd()
			if err != nil {
				util.Die(err)
			}
			dir = filepath.Base(dir)
		}

		stackName = dir
	}

	fmt.Printf("Deploying %s => %s\n", filepath.Base(fn), stackName)

	if stackExists(stackName) {
		fmt.Println("Stack exists. Showing diff:")

		template := cfn.GetStackTemplate(stackName)

		left, err := parse.ReadString(template)
		if err != nil {
			util.Die(err)
		}

		right, err := parse.ReadFile(fn)
		if err != nil {
			util.Die(err)
		}

		fmt.Print(diff.Format(diff.Compare(left, right)))
	}

	fmt.Println("TODO: CONFIRM")
}

func stackExists(stackName string) bool {
	ch := make(chan bool)

	go func() {
		cfn.ListStacks(func(s cloudformation.StackSummary) {
			if *s.StackName == stackName {
				ch <- true
			}
		})

		// Default
		ch <- false
	}()

	return <-ch
}
