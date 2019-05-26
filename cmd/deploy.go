package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/util"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:                   "deploy [template] [stack]",
	Short:                 "Deploy a CloudFormation stack",
	Long:                  "Creates or updates a CloudFormation stack named [stack] from the template file [template].",
	Args:                  cobra.ExactArgs(2),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fn := args[0]
		stackName := args[1]

		fmt.Printf("Deploying %s => %s\n", filepath.Base(fn), stackName)

		outputFn, err := ioutil.TempFile("", "")
		if err != nil {
			util.Die(err)
		}
		fmt.Println(outputFn.Name())
		defer os.Remove(outputFn.Name())

		// Package it up
		_, err = util.RunCapture("aws", "cloudformation", "package",
			"--template-file", fn,
			"--output-template-file", outputFn.Name(),
			"--s3-bucket", getRainBucket(),
		)
		if err != nil {
			util.Die(err)
		}

		// Start deployment
		cfn.Deploy(outputFn.Name(), stackName)
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
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
