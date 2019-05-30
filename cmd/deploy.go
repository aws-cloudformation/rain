package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/diff"
	"github.com/aws-cloudformation/rain/util"
	"github.com/awslabs/aws-cloudformation-template-formatter/parse"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:                   "deploy <template> <stack>",
	Short:                 "Deploy a CloudFormation stack from a local template",
	Long:                  "Creates or updates a CloudFormation stack named <stack> from the template file <template>.",
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

		// Find out if stack exists already
		// If it does and it's not in a good state, offer to wait/delete
		stack, err := cfn.GetStack(stackName)
		if err == nil {
			if !strings.HasSuffix(string(stack.StackStatus), "_COMPLETE") {
				// Can't update
				util.Die(fmt.Errorf("Stack '%s' could not be updated: %s", stackName, colouriseStatus(string(stack.StackStatus))))
			} else {
				// Can update, grab a diff
				fmt.Println("Stack exists. Diff will follow...")

				oldTemplateString := cfn.GetStackTemplate(stackName)

				oldTemplate, _ := parse.ReadString(oldTemplateString)
				newTemplate, _ := parse.ReadFile(outputFn.Name())

				d := diff.Compare(oldTemplate, newTemplate)

				if d == diff.Unchanged {
					util.Die(errors.New("No changes to deploy!"))
				}

				fmt.Println(colouriseDiff(d))

				fmt.Println("Stack exists; see above diff")

				for i := 5; i >= 0; i-- {
					fmt.Printf("Continuting in %d seconds...", i)
					time.Sleep(time.Second)
					fmt.Print("\033[1G")
				}
			}
		}

		// Start deployment
		cfn.Deploy(outputFn.Name(), stackName)
		cfn.WaitUntilStackExists(stackName)

		status := waitForStackToSettle(stackName)

		if status == "CREATE_COMPLETE" {
			fmt.Println(util.Text{"Successfully deployed " + stackName, util.Green})
		} else if status == "UPDATE_COMPLETE" {
			fmt.Println(util.Text{"Successfully updated " + stackName, util.Green})
		} else {
			fmt.Println(util.Text{"Failed deployment: " + stackName, util.Red})
		}

		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
