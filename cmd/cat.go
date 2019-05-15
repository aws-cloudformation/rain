package cmd

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
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

	req := cfnClient.GetTemplateRequest(&cloudformation.GetTemplateInput{
		StackName: &args[0],
	})

	res, err := req.Send()
	if err != nil {
		panic(err)
	}

	fmt.Println(*res.TemplateBody)
}
