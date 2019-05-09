package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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
	var err error

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: rain deploy <template> [stack]")
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
				panic(err)
			}
			dir = filepath.Base(dir)
		}

		stackName = dir
	}

	fmt.Printf("Deploying %s => %s\n", filepath.Base(fn), stackName)

	if stackExists(stackName) {
		stackTemplateFn, err := ioutil.TempFile("", "")
		if err != nil {
			panic(err)
		}

		out, err := util.RunCapture(
			"aws",
			"cloudformation",
			"get-template",
			"--query", "TemplateBody",
			"--stack-name", stackName,
		)

		if err != nil {
			panic(err)
		}

		ioutil.WriteFile(stackTemplateFn.Name(), []byte(out), 0600)

		fmt.Println(stackTemplateFn.Name())

		diffCommand([]string{fn, stackTemplateFn.Name()})

		err = os.Remove(stackTemplateFn.Name())
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("TODO: CONFIRM")
}

func stackExists(stackName string) bool {
	cmdArgs := append([]string{
		"cloudformation",
		"list-stacks",
		"--output", "text",
		"--query", "StackSummaries[].[StackName]",
		"--stack-status-filter",
	}, liveStatuses...)

	out, err := util.RunCapture("aws", cmdArgs...)

	if err != nil {
		panic(err)
	}

	for _, name := range strings.Split(out, "\n") {
		if name == stackName {
			return true
		}
	}

	return false
}
