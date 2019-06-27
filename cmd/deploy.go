package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/diff"
	"github.com/aws-cloudformation/rain/parse"
	"github.com/aws-cloudformation/rain/util"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/spf13/cobra"
)

func getParameters(t string, old []cloudformation.Parameter, forceOldValue bool) []cloudformation.Parameter {
	newParams := make([]cloudformation.Parameter, 0)

	template, err := parse.ReadString(t)
	if err != nil {
		panic(fmt.Errorf("Unable to parse template: %s", err))
	}

	oldMap := make(map[string]cloudformation.Parameter)
	for _, param := range old {
		oldMap[*param.ParameterKey] = param
	}

	if params, ok := template["Parameters"]; ok {
		for k, p := range params.(map[string]interface{}) {
			// New variable so we don't mess up the pointers below
			key := k

			extra := ""
			param := p.(map[string]interface{})

			hasExisting := false

			value := ""

			if oldParam, ok := oldMap[key]; ok {
				extra = fmt.Sprintf(" (existing value: %s)", *oldParam.ParameterValue)
				hasExisting = true

				if forceOldValue {
					value = *oldParam.ParameterValue
				}
			} else if defaultValue, ok := param["Default"]; ok {
				extra = fmt.Sprintf(" (default value: %s)", defaultValue)
			}

			newValue := util.Ask(fmt.Sprintf("Enter a value for parameter '%s'%s:", key, extra))

			if newValue != "" {
				newParams = append(newParams, cloudformation.Parameter{
					ParameterKey:   &key,
					ParameterValue: &newValue,
				})
			} else if value != "" && forceOldValue {
				newParams = append(newParams, cloudformation.Parameter{
					ParameterKey:   &key,
					ParameterValue: &value,
				})
			} else if hasExisting {
				newParams = append(newParams, cloudformation.Parameter{
					ParameterKey:     &key,
					UsePreviousValue: &hasExisting,
				})
			}
		}
	}

	return newParams
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

		fmt.Printf("Deploying '%s' as '%s' in %s:\n", filepath.Base(fn), stackName, client.Config().Region)

		fmt.Print("Preparing template... ")

		outputFn, err := ioutil.TempFile("", "")
		if err != nil {
			panic(err)
		}
		defer func() {
			util.Debug("Removing temporary template file: %s", outputFn.Name())
			err := os.Remove(outputFn.Name())
			if err != nil {
				panic(fmt.Errorf("Error removing temporary template file '%s': %s", outputFn.Name(), err))
			}
		}()

		output, err := util.RunAwsCapture("cloudformation", "package",
			"--template-file", fn,
			"--output-template-file", outputFn.Name(),
			"--s3-bucket", getRainBucket(),
		)
		if err != nil {
			panic(fmt.Errorf("Unable to package template: %s", err))
		}

		util.Debug("Package output: %s", output)

		util.ClearLine()
		fmt.Printf("Checking current status of stack '%s'... ", stackName)

		forceOldParams := false

		// Find out if stack exists already
		// If it does and it's not in a good state, offer to wait/delete
		stack, err := cfn.GetStack(stackName)
		if err == nil {
			if string(stack.StackStatus) == "ROLLBACK_COMPLETE" {
				forceOldParams = true

				fmt.Println("Stack is currently ROLLBACK_COMPLETE; deleting...")
				err := cfn.DeleteStack(stackName)
				if err != nil {
					panic(fmt.Errorf("Unable to delete stack '%s': %s", stackName, err))
				}

				status := waitForStackToSettle(stackName)

				if status != "DELETE_COMPLETE" {
					panic(fmt.Errorf("Failed to delete " + stackName))
				}
			} else if !strings.HasSuffix(string(stack.StackStatus), "_COMPLETE") {
				// Can't update
				panic(fmt.Errorf("Stack '%s' could not be updated: %s", stackName, colouriseStatus(string(stack.StackStatus))))
			} else {
				// Can update, grab a diff

				oldTemplateString, err := cfn.GetStackTemplate(stackName)
				if err != nil {
					panic(fmt.Errorf("Failed to get existing template for stack '%s': %s", stackName, err))
				}

				oldTemplate, _ := parse.ReadString(oldTemplateString)
				newTemplate, _ := parse.ReadFile(outputFn.Name())

				d := diff.Compare(oldTemplate, newTemplate)

				if d.Mode() == diff.Unchanged {
					fmt.Println(util.Green("No changes to deploy!"))
					return
				}

				util.ClearLine()
				if util.Confirm(true, fmt.Sprintf("Stack '%s' exists. Do you wish to see the diff before deploying?", stackName)) {
					fmt.Print(colouriseDiff(d, false))

					if !util.Confirm(true, "Do you wish to continue?") {
						panic(errors.New("User cancelled deployment."))
					}
				}
			}
		}

		// Load in the template file
		t, err := ioutil.ReadFile(outputFn.Name())
		if err != nil {
			panic(fmt.Errorf("Can't load template '%s': %s", outputFn.Name(), err))
		}
		template := string(t)

		parameters := getParameters(template, stack.Parameters, forceOldParams)

		util.Debug("Parameters: %s", parameters)

		// Start deployment
		err = cfn.Deploy(template, parameters, stackName)
		if err != nil {
			panic(fmt.Errorf("Error during deployment of '%s': %s", stackName, err))
		}

		err = cfn.WaitUntilStackExists(stackName)
		if err != nil {
			panic(fmt.Errorf("Error getting stack status '%s': %s", stackName, err))
		}

		status := waitForStackToSettle(stackName)

		if status == "CREATE_COMPLETE" {
			fmt.Println(util.Green("Successfully deployed " + stackName))
		} else if status == "UPDATE_COMPLETE" {
			fmt.Println(util.Green("Successfully updated " + stackName))
		} else {
			panic(errors.New("Failed deployment: " + stackName))
		}

		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
