package deploy

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
)

var fixStackNameRe *regexp.Regexp

const maxStackNameLength = 128

func formatChangeSet(status *cloudformation.DescribeChangeSetOutput) string {
	out := strings.Builder{}

	out.WriteString(fmt.Sprintf("%s:\n", console.Yellow(fmt.Sprintf("Stack %s", ptr.ToString(status.StackName)))))

	for _, change := range status.Changes {
		line := fmt.Sprintf("%s %s",
			*change.ResourceChange.ResourceType,
			*change.ResourceChange.LogicalResourceId,
		)

		switch change.ResourceChange.Action {
		case types.ChangeAction("Add"):
			out.WriteString(console.Green("  + " + line))
		case types.ChangeAction("Modify"):
			out.WriteString(console.Blue("  > " + line))
		case types.ChangeAction("Remove"):
			out.WriteString(console.Red("  - " + line))
		}

		out.WriteString("\n")
	}

	return strings.TrimSpace(out.String())
}

func getParameters(template cft.Template, cliParams map[string]string, old []types.Parameter, stackExists bool) []types.Parameter {
	newParams := make([]types.Parameter, 0)

	oldMap := make(map[string]types.Parameter)
	for _, param := range old {
		// Ignore NoEcho values
		if stackExists || ptr.ToString(param.ParameterValue) != "****" {
			oldMap[ptr.ToString(param.ParameterKey)] = param
		}
	}

	if params, ok := template.Map()["Parameters"]; ok {
		// Check we don't have any unknown params
		for k := range cliParams {
			if _, ok := params.(map[string]interface{})[k]; !ok {
				panic(fmt.Errorf("unknown parameter: %s", k))
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
				} else if yes {
					panic(fmt.Errorf("no default or existing value for parameter '%s'. Set a default, supply a --params flag, or deploy without the --yes flag", k))
				}

				if !yes {
					spinner.Pause()

					prompt := fmt.Sprintf("Enter a value for parameter '%s'", k)

					if description, ok := param["Description"]; ok {
						prompt += fmt.Sprintf(" \"%s\"", description)
					}

					prompt += fmt.Sprintf("%s:", extra)

					newValue := console.Ask(prompt)
					if newValue != "" {
						value = newValue
						usePrevious = false
					}
				}
			}

			if usePrevious {
				newParams = append(newParams, types.Parameter{
					ParameterKey:     ptr.String(k),
					UsePreviousValue: ptr.Bool(true),
				})
			} else {
				newParams = append(newParams, types.Parameter{
					ParameterKey:   ptr.String(k),
					ParameterValue: ptr.String(value),
				})
			}
		}
	}

	spinner.Resume()

	return newParams
}

// ListToMap converts a pflag parsed StringSlice into a map
// where values are expected to be presented in the form
// Foo=bar,Baz=quux,mooz,Xyzzy=garply
func ListToMap(name string, in []string) map[string]string {
	out := make(map[string]string, len(in))
	lastKey := ""
	for _, v := range in {
		parts := strings.SplitN(v, "=", 2)

		if len(parts) != 2 {
			if lastKey == "" {
				panic(fmt.Errorf("unable to parse %s: %s", name, v))
			} else {
				out[lastKey] += "," + parts[0]
			}
		} else {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			if _, ok := out[key]; ok {
				panic(fmt.Errorf("duplicate %s: %s", name, key))
			}

			out[key] = value

			lastKey = key
		}
	}

	return out
}

func packageTemplate(fn string, yes bool) cft.Template {
	// Call RainBucket for side-effects in case we want to force bucket creation
	s3.RainBucket(yes)

	t, err := parse.File(fn)
	if err != nil {
		panic(ui.Errorf(err, "error reading template file '%s'", fn))
	}

	t, err = pkg.Template(t)
	if err != nil {
		panic(ui.Errorf(err, "error packaging template '%s'", fn))
	}

	return t
}

func checkStack(stackName string) (types.Stack, bool) {
	// Find out if stack exists already
	// If it does and it's not in a good state, offer to wait/delete
	stack, err := cfn.GetStack(stackName)

	stackExists := false
	if err == nil {
		config.Debugf("Stack exists")
		stackExists = true
	}

	spinner.Pause()

	if stackExists {
		switch {
		case stack.StackStatus == types.StackStatusRollbackComplete,
			stack.StackStatus == types.StackStatusReviewInProgress,
			stack.StackStatus == types.StackStatusCreateFailed:

			message := "Existing stack is empty; deleting it."
			fmt.Println(message)

			err := cfn.DeleteStack(stackName)
			if err != nil {
				panic(ui.Errorf(err, "unable to delete stack '%s'", stackName))
			}

			status, _ := ui.WaitForStackToSettle(stackName)

			if status != "DELETE_COMPLETE" {
				panic(fmt.Errorf("failed to delete stack '%s'", stackName))
			}

			console.ClearLines(console.CountLines(message) + 1)
			fmt.Println("Deleted existing, empty stack.")

			stackExists = false
		case !strings.HasSuffix(string(stack.StackStatus), "_COMPLETE"):
			// Can't update
			panic(fmt.Errorf("stack '%s' could not be updated: %s", stackName, ui.ColouriseStatus(string(stack.StackStatus))))
		}
	}

	spinner.Resume()

	return stack, stackExists
}

func init() {
	fixStackNameRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)
}
