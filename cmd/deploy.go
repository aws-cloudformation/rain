package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/util"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [template file] [stack name]",
	Short: "Deploy templates to stacks",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fn := args[0]
		stackName := args[1]

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
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
