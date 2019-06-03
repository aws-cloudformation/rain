package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/diff"
	"github.com/aws-cloudformation/rain/parse"
	"github.com/aws-cloudformation/rain/util"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/spf13/cobra"
)

func getParameters(t string, old []cloudformation.Parameter) []cloudformation.Parameter {
	new := make([]cloudformation.Parameter, 0)

	template, err := parse.ReadString(t)
	if err != nil {
		util.Die(err)
	}

	oldMap := make(map[string]cloudformation.Parameter)
	for _, param := range old {
		oldMap[*param.ParameterKey] = param
	}

	if params, ok := template["Parameters"]; ok {
		for key, p := range params.(map[string]interface{}) {
			extra := ""
			param := p.(map[string]interface{})

			hasExisting := false

			if oldParam, ok := oldMap[key]; ok {
				extra = fmt.Sprintf(" (existing value: %s)", *oldParam.ParameterValue)
				hasExisting = true
			} else if defaultValue, ok := param["Default"]; ok {
				extra = fmt.Sprintf(" (default value: %s)", defaultValue)
			}

			value := util.Ask(fmt.Sprintf("Enter a value for parameter '%s'%s:", key, extra))

			if value != "" {
				new = append(new, cloudformation.Parameter{
					ParameterKey:   &key,
					ParameterValue: &value,
				})
			} else if hasExisting {
				new = append(new, cloudformation.Parameter{
					ParameterKey:     &key,
					UsePreviousValue: &hasExisting,
				})
			}
		}
	}

	return new
}

var deployCmd = &cobra.Command{
	Use:                   "deploy <template> <stack>",
	Short:                 "Deploy a CloudFormation stack from a local template",
	Long:                  "Creates or updates a CloudFormation stack named <stack> from the template file <template>.",
	Args:                  cobra.ExactArgs(2),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fn := args[0]
		stackName := args[1]

		fmt.Printf("Preparing template '%s'... ", filepath.Base(fn))

		outputFn, err := ioutil.TempFile("", "")
		if err != nil {
			util.Die(err)
		}
		defer os.Remove(outputFn.Name())

		_, err = util.RunAwsCapture("cloudformation", "package",
			"--template-file", fn,
			"--output-template-file", outputFn.Name(),
			"--s3-bucket", getRainBucket(),
		)
		if err != nil {
			util.Die(err)
		}

		util.ClearLine()
		fmt.Printf("Checking current status of stack '%s'... ", stackName)

		// Find out if stack exists already
		// If it does and it's not in a good state, offer to wait/delete
		stack, err := cfn.GetStack(stackName)
		if err == nil {
			if !strings.HasSuffix(string(stack.StackStatus), "_COMPLETE") {
				// Can't update
				util.Die(fmt.Errorf("Stack '%s' could not be updated: %s", stackName, colouriseStatus(string(stack.StackStatus))))
			} else {
				// Can update, grab a diff

				oldTemplateString, err := cfn.GetStackTemplate(stackName)
				if err != nil {
					util.Die(fmt.Errorf("Failed to get existing template for stack '%s': %s", stackName, err))
				}

				oldTemplate, _ := parse.ReadString(oldTemplateString)
				newTemplate, _ := parse.ReadFile(outputFn.Name())

				d := diff.Compare(oldTemplate, newTemplate)

				if d == diff.Unchanged {
					fmt.Println(util.Green("No changes to deploy!"))
					return
				}

				util.ClearLine()
				if util.Confirm(true, fmt.Sprintf("Stack '%s' exists. Do you wish to see the diff before deploying?", stackName)) {
					fmt.Print(colouriseDiff(d))

					if !util.Confirm(true, "Do you wish to continue?") {
						util.Die(errors.New("User cancelled deployment."))
					}
				}
			}
		}

		util.ClearLine()
		fmt.Printf("Deploying stack '%s'... ", stackName)

		// Load in the template file
		t, err := ioutil.ReadFile(outputFn.Name())
		if err != nil {
			util.Die(err)
		}
		template := string(t)

		parameters := getParameters(template, stack.Parameters)

		// Start deployment
		err = cfn.Deploy(template, parameters, stackName)
		if err != nil {
			util.Die(err)
		}

		err = cfn.WaitUntilStackExists(stackName)
		if err != nil {
			util.Die(err)
		}

		status := waitForStackToSettle(stackName)

		if status == "CREATE_COMPLETE" {
			fmt.Println(util.Green("Successfully deployed " + stackName))
		} else if status == "UPDATE_COMPLETE" {
			fmt.Println(util.Green("Successfully updated " + stackName))
		} else {
			util.Die(errors.New("Failed deployment: " + stackName))
		}

		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
