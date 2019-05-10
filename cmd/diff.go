package cmd

import (
	"fmt"
	"os"

	"github.com/aws-cloudformation/rain/diff"
	"github.com/awslabs/aws-cloudformation-template-formatter/parse"
)

func init() {
	Commands["diff"] = Command{
		Type: TEMPLATE,
		Run:  diffCommand,
		Help: "Compare templates with other templates or stacks",
	}
}

func diffCommand(args []string) {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: rain diff <left> <right>")
		os.Exit(1)
	}

	leftFn, rightFn := args[0], args[1]

	left, err := parse.ReadFile(leftFn)
	if err != nil {
		panic(err)
	}

	right, err := parse.ReadFile(rightFn)
	if err != nil {
		panic(err)
	}

	fmt.Print(diff.Format(diff.Compare(left, right)))
}
