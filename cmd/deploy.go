package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/cfn/diff"
	"github.com/aws-cloudformation/rain/cfn/parse"
	"github.com/aws-cloudformation/rain/client"
	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/config"
	"github.com/aws-cloudformation/rain/console"
	"github.com/aws-cloudformation/rain/console/spinner"
	"github.com/aws-cloudformation/rain/console/text"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/spf13/cobra"
)

var force = false

func formatChangeSet(changes []cloudformation.Change) string {
	out := strings.Builder{}

	for _, change := range changes {
		line := fmt.Sprintf("%s %s\n",
			*change.ResourceChange.ResourceType,
			*change.ResourceChange.LogicalResourceId,
		)

		switch change.ResourceChange.Action {
		case cloudformation.ChangeAction("Add"):
			out.WriteString(text.Green("(+) " + line).String())
		case cloudformation.ChangeAction("Modify"):
			out.WriteString(text.Orange("(|) " + line).String())
		case cloudformation.ChangeAction("Remove"):
			out.WriteString(text.Red("(-) " + line).String())
		}
	}

	return out.String()
}

func getParameters(t string, old []cloudformation.Parameter, forceOldValue bool) []cloudformation.Parameter {
	newParams := make([]cloudformation.Parameter, 0)

	template, err := parse.String(t)
	if err != nil {
		panic(fmt.Errorf("Unable to parse template: %s", err))
	}

	oldMap := make(map[string]cloudformation.Parameter)
	for _, param := range old {
		oldMap[*param.ParameterKey] = param
	}

	if params, ok := template.Map()["Parameters"]; ok {
		for k, p := range params.(map[string]interface{}) {
			// New variable so we don't mess up the pointers below
			key := k

			extra := ""
			param := p.(map[string]interface{})

			hasExisting := false

			value := ""

			if oldParam, ok := oldMap[key]; ok {
				extra = fmt.Sprintf(" (existing value: %s)", fmt.Sprint(*oldParam.ParameterValue))
				hasExisting = true

				if forceOldValue {
					value = *oldParam.ParameterValue
				}
			} else if defaultValue, ok := param["Default"]; ok {
				extra = fmt.Sprintf(" (default value: %s)", fmt.Sprint(defaultValue))
			}

			if force {
				panic(fmt.Errorf("Some parameters require values. Set defaults or deploy without the --force flag."))
			}

			newValue := console.Ask(fmt.Sprintf("Enter a value for parameter '%s'%s:", key, extra))

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
			config.Debugf("Removing temporary template file: %s", outputFn.Name())
			err := os.Remove(outputFn.Name())
			if err != nil {
				panic(fmt.Errorf("Error removing temporary template file '%s': %s", outputFn.Name(), err))
			}
		}()

		output, err := runAws("cloudformation", "package",
			"--template-file", fn,
			"--output-template-file", outputFn.Name(),
			"--s3-bucket", getRainBucket(),
		)
		if err != nil {
			panic(fmt.Errorf("Unable to package template: %s", err))
		}

		config.Debugf("Package output: %s", output)

		console.ClearLine()
		fmt.Printf("Checking current status of stack '%s'... ", stackName)

		forceOldParams := false

		// Find out if stack exists already
		// If it does and it's not in a good state, offer to wait/delete
		stack, err := cfn.GetStack(stackName)
		stackExists := false
		if err == nil {
			stackExists = true
		}

		if stackExists {
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
			} else if !force {
				// Can update, grab a diff

				oldTemplateString, err := cfn.GetStackTemplate(stackName, false)
				if err != nil {
					panic(fmt.Errorf("Failed to get existing template for stack '%s': %s", stackName, err))
				}

				oldTemplate, _ := parse.String(oldTemplateString)
				newTemplate, _ := parse.File(outputFn.Name())

				d := oldTemplate.Diff(newTemplate)

				if d.Mode() == diff.Unchanged {
					fmt.Println(text.Green("No changes to deploy!"))
					return
				}

				console.ClearLine()
				if console.Confirm(true, fmt.Sprintf("Stack '%s' exists. Do you wish to compare the CloudFormation templates?", stackName)) {
					fmt.Print(colouriseDiff(d, false))
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

		config.Debugf("Parameters: %s", parameters)

		// Create a change set
		spinner.Status("Creating change set...")
		changeSetName, err := cfn.CreateChangeSet(template, parameters, stackName)
		if err != nil {
			panic(fmt.Errorf("Error while creating changeset for '%s': %s", stackName, err))
		}
		changes, err := cfn.GetChangeSet(stackName, changeSetName)
		if err != nil {
			panic(fmt.Errorf("Error while retrieving changeset '%s': %s", changeSetName, err))
		}
		spinner.Stop()

		fmt.Println("CloudFormation will make the following changes:")
		fmt.Println(formatChangeSet(changes))

		if !force && !console.Confirm(true, "Do you wish to continue?") {
			err = cfn.DeleteChangeSet(stackName, changeSetName)
			if err != nil {
				panic(fmt.Errorf("Error while deleting changeset '%s': %s", changeSetName, err))
			}

			if !stackExists {
				err = cfn.DeleteStack(stackName)
				if err != nil {
					panic(fmt.Errorf("Error deleting empty stack '%s': %s", stackName, err))
				}
			}

			panic(errors.New("User cancelled deployment."))
		} else {
			err = cfn.ExecuteChangeSet(stackName, changeSetName)
			if err != nil {
				panic(fmt.Errorf("Error while executing changeset '%s': %s", changeSetName, err))
			}
		}

		status := waitForStackToSettle(stackName)

		if status == "CREATE_COMPLETE" {
			fmt.Println(text.Green("Successfully deployed " + stackName))
		} else if status == "UPDATE_COMPLETE" {
			fmt.Println(text.Green("Successfully updated " + stackName))
		} else {
			panic(errors.New("Failed deployment: " + stackName))
		}

		fmt.Println()
	},
}

func init() {
	deployCmd.Flags().BoolVarP(&force, "force", "f", false, "Don't ask questions; just deploy.")
	Root.AddCommand(deployCmd)
}
