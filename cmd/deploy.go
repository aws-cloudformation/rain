package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	cft "github.com/aws-cloudformation/rain/cfn"
	"github.com/aws-cloudformation/rain/cfn/diff"
	"github.com/aws-cloudformation/rain/cfn/parse"
	"github.com/aws-cloudformation/rain/client"
	"github.com/aws-cloudformation/rain/client/cfn"
	"github.com/aws-cloudformation/rain/config"
	"github.com/aws-cloudformation/rain/console"
	"github.com/aws-cloudformation/rain/console/spinner"
	"github.com/aws-cloudformation/rain/console/text"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/cobra"
)

var detachDeploy bool
var forceDeploy bool
var params []string
var tags []string

var fixStackNameRe *regexp.Regexp

const maxStackNameLength = 128

func formatChangeSet(status *cloudformation.DescribeChangeSetOutput) string {
	out := strings.Builder{}

	out.WriteString(fmt.Sprintf("Stack \"%s\": %s\n", aws.ToString(status.StackName), aws.ToString(status.StatusReason)))

	for _, change := range status.Changes {
		line := fmt.Sprintf("%s %s\n",
			*change.ResourceChange.ResourceType,
			*change.ResourceChange.LogicalResourceId,
		)

		switch change.ResourceChange.Action {
		case types.ChangeAction("Add"):
			out.WriteString(text.Green("(+) " + line).String())
		case types.ChangeAction("Modify"):
			out.WriteString(text.Orange("(|) " + line).String())
		case types.ChangeAction("Remove"):
			out.WriteString(text.Red("(-) " + line).String())
		}
	}

	return out.String()
}

func getParameters(template cft.Template, cliParams map[string]string, old []*types.Parameter, stackExists bool) []*types.Parameter {
	newParams := make([]*types.Parameter, 0)

	oldMap := make(map[string]*types.Parameter)
	for _, param := range old {
		oldMap[*param.ParameterKey] = param
	}

	if params, ok := template.Map()["Parameters"]; ok {
		// Check we don't have any unknown params
		for k := range cliParams {
			if _, ok := params.(map[string]interface{})[k]; !ok {
				panic(fmt.Errorf("Unknown parameter: %s", k))
			}
		}

		// Decide on a default value
		for k, p := range params.(map[string]interface{}) {
			// New variable so we don't mess up the pointers below
			param := p.(map[string]interface{})

			value := ""
			usePrevious := false

			// Decide if we have an existing value
			if cliParam, ok := cliParams[k]; ok {
				value = cliParam
			} else {
				extra := ""

				if oldParam, ok := oldMap[k]; ok {
					extra = fmt.Sprintf(" (existing value: %s)", fmt.Sprint(*oldParam.ParameterValue))

					if stackExists {
						usePrevious = true
					} else {
						value = *oldParam.ParameterValue
					}
				} else if defaultValue, ok := param["Default"]; ok {
					extra = fmt.Sprintf(" (default value: %s)", fmt.Sprint(defaultValue))
					value = fmt.Sprint(defaultValue)
				} else if forceDeploy {
					panic(fmt.Errorf("No default or existing value for parameter '%s'. Set a default, supply a --param flag, or deploy without the --force flag", k))
				}

				if !forceDeploy {
					newValue := console.Ask(fmt.Sprintf("Enter a value for parameter '%s'%s:", k, extra))
					if newValue != "" {
						value = newValue
						usePrevious = false
					}
				}
			}

			if usePrevious {
				newParams = append(newParams, &types.Parameter{
					ParameterKey:     aws.String(k),
					UsePreviousValue: aws.Bool(true),
				})
			} else {
				newParams = append(newParams, &types.Parameter{
					ParameterKey:   aws.String(k),
					ParameterValue: aws.String(value),
				})
			}
		}
	}

	return newParams
}

func listToMap(name string, in []string) map[string]string {
	out := make(map[string]string, len(in))
	for _, v := range in {
		parts := strings.SplitN(v, "=", 2)

		if len(parts) != 2 {
			panic(fmt.Errorf("Unable to parse %s: %s", name, v))
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if _, ok := out[key]; ok {
			panic(fmt.Errorf("Duplicate %s: %s", name, key))
		}

		out[key] = value
	}

	return out
}

var deployCmd = &cobra.Command{
	Use:   "deploy <template> [stack]",
	Short: "Deploy a CloudFormation stack from a local template",
	Long: `Creates or updates a CloudFormation stack named <stack> from the template file <template>.
If you don't specify a stack name, rain will use the template filename minus its extension.

If a template needs to be packaged before it can be deployed, rain will package the template first.
Rain will attempt to create an S3 bucket to store artifacts that it packages and deploys.
The bucket's name will be of the format rain-artifacts-<AWS account id>-<AWS region>`,
	Args:                  cobra.RangeArgs(1, 2),
	Annotations:           stackAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fn := args[0]

		var stackName string

		if len(args) == 2 {
			stackName = args[1]
		} else {
			base := path.Base(fn)
			stackName = base[:len(base)-len(path.Ext(base))]

			// Now ensure it's a valid cfn name
			stackName = fixStackNameRe.ReplaceAllString(stackName, "-")

			if len(stackName) > maxStackNameLength {
				stackName = stackName[:maxStackNameLength]
			}
		}

		// Parse tags
		parsedTags := listToMap("tag", tags)

		// Parse params
		parsedParams := listToMap("param", params)

		fmt.Printf("Deploying '%s' as '%s' in %s:\n", filepath.Base(fn), stackName, client.Config().Region)

		spinner.Status("Preparing template")

		outputFn, err := ioutil.TempFile("", "")
		if err != nil {
			panic(err)
		}
		defer func() {
			config.Debugf("Removing temporary template file: %s", outputFn.Name())
			err := os.Remove(outputFn.Name())
			if err != nil {
				panic(errorf(err, "Error removing temporary template file '%s'", outputFn.Name()))
			}
		}()

		output, err := runAws("cloudformation", "package",
			"--template-file", fn,
			"--output-template-file", outputFn.Name(),
			"--s3-bucket", getRainBucket(),
		)
		if err != nil {
			panic(errorf(err, "Unable to package template"))
		}

		config.Debugf("Package output: %s", output)

		// Load in the packagedctemplate
		config.Debugf("Loading packaged template file")
		template, err := parse.File(outputFn.Name())
		if err != nil {
			panic(errorf(err, "Error reading packaged template '%s'", outputFn.Name()))
		}

		spinner.Stop()

		spinner.Status(fmt.Sprintf("Checking current status of stack '%s'... ", stackName))

		// Find out if stack exists already
		// If it does and it's not in a good state, offer to wait/delete
		stack, err := cfn.GetStack(stackName)

		stackExists := false
		if err == nil {
			config.Debugf("Stack exists")
			stackExists = true
		}

		spinner.Stop()

		if stackExists {
			if string(stack.StackStatus) == "ROLLBACK_COMPLETE" {
				fmt.Println("Stack is currently ROLLBACK_COMPLETE; deleting...")
				err := cfn.DeleteStack(stackName)
				if err != nil {
					panic(errorf(err, "Unable to delete stack '%s'", stackName))
				}

				status := waitForStackToSettle(stackName)

				if status != "DELETE_COMPLETE" {
					panic(fmt.Errorf("Failed to delete " + stackName))
				}

				stackExists = false
			} else if !strings.HasSuffix(string(stack.StackStatus), "_COMPLETE") {
				// Can't update
				panic(fmt.Errorf("Stack '%s' could not be updated: %s", stackName, colouriseStatus(string(stack.StackStatus))))
			} else if !forceDeploy {
				// Can update, grab a diff

				oldTemplateString, err := cfn.GetStackTemplate(stackName, false)
				if err != nil {
					panic(errorf(err, "Failed to get existing template for stack '%s'", stackName))
				}

				oldTemplate, _ := parse.String(oldTemplateString)

				d := oldTemplate.Diff(template)

				if d.Mode() != diff.Unchanged {
					console.ClearLine()
					if console.Confirm(true, fmt.Sprintf("Stack '%s' exists. Do you wish to compare the CloudFormation templates?", stackName)) {
						fmt.Print(colouriseDiff(d, false))
					}
				}
			}
		}

		config.Debugf("Handling parameters")
		parameters := getParameters(template, parsedParams, stack.Parameters, stackExists)

		if config.Debug {
			for _, param := range parameters {
				config.Debugf("  %s: %s", *param.ParameterKey, *param.ParameterValue)
			}
		}

		// Create a change set
		spinner.Status("Creating change set")
		changeSetName, createErr := cfn.CreateChangeSet(template, parameters, parsedTags, stackName)
		if createErr != nil {
			panic(errorf(createErr, "Error creating changeset"))
		}

		changeSetStatus, err := cfn.GetChangeSet(stackName, changeSetName)
		if err != nil {
			panic(fmt.Errorf("Error getting changeset status: %s", formatChangeSet(changeSetStatus)))
		}

		spinner.Stop()

		if !forceDeploy {
			fmt.Println("CloudFormation will make the following changes:")
			fmt.Println(formatChangeSet(changeSetStatus))

			if !console.Confirm(true, "Do you wish to continue?") {
				err = cfn.DeleteChangeSet(stackName, changeSetName)
				if err != nil {
					panic(errorf(err, "Error while deleting changeset '%s'", changeSetName))
				}

				if !stackExists {
					err = cfn.DeleteStack(stackName)
					if err != nil {
						panic(errorf(err, "Error deleting empty stack '%s'", stackName))
					}
				}

				panic(errors.New("User cancelled deployment"))
			}
		}

		// Deploy!
		err = cfn.ExecuteChangeSet(stackName, changeSetName)
		if err != nil {
			panic(errorf(err, "Error while executing changeset '%s'", changeSetName))
		}

		if detachDeploy {
			fmt.Printf("Detaching. You can check your stack's status with: rain watch %s\n", stackName)
		} else {
			status := waitForStackToSettle(stackName)

			stack, _ = cfn.GetStack(stackName)
			console.Clear(getStackOutput(stack, false))

			if status == "CREATE_COMPLETE" {
				fmt.Println(text.Green("Successfully deployed " + stackName))
			} else if status == "UPDATE_COMPLETE" {
				fmt.Println(text.Green("Successfully updated " + stackName))
			} else {
				logsCmd.Run(Rain, []string{stackName})
				panic(errors.New("Failed deployment: " + stackName))
			}
		}

		fmt.Println()
	},
}

func init() {
	fixStackNameRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

	deployCmd.Flags().BoolVarP(&detachDeploy, "detach", "d", false, "Once deployment has started, don't wait around for it to finish.")
	deployCmd.Flags().BoolVarP(&forceDeploy, "force", "f", false, "Don't ask questions; just deploy.")
	deployCmd.Flags().StringSliceVar(&tags, "tags", []string{}, "Add tags to the stack. Use the format key1=value1,key2=value2.")
	deployCmd.Flags().StringSliceVar(&params, "params", []string{}, "Set parameter values. Use the format key1=value1,key2=value2.")
	Rain.AddCommand(deployCmd)
}
