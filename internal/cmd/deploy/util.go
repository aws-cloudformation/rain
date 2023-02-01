package deploy

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
)

var FixStackNameRe *regexp.Regexp

const MaxStackNameLength = 128

func formatChangeSet(stackName, changeSetName string) string {
	status, err := cfn.GetChangeSet(stackName, changeSetName)
	if err != nil {
		panic(ui.Errorf(err, "error getting changeset '%s' for stack '%s'", changeSetName, stackName))
	}

	out := strings.Builder{}

	out.WriteString(fmt.Sprintf("%s:\n", console.Yellow(fmt.Sprintf("Stack %s", ptr.ToString(status.StackName)))))

	// Non-stack resources
	for _, change := range status.Changes {
		if change.ResourceChange.ChangeSetId != nil {
			// Bunch up nested stacks to the end
			continue
		}

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

	// Nested stacks
	for _, change := range status.Changes {
		if change.ResourceChange.ChangeSetId == nil {
			continue
		}

		child := formatChangeSet("", ptr.ToString(change.ResourceChange.ChangeSetId))
		parts := strings.SplitN(child, "\n", 2)
		header, body := parts[0], parts[1]

		switch change.ResourceChange.Action {
		case types.ChangeAction("Add"):
			out.WriteString(console.Green("  + " + header))
		case types.ChangeAction("Modify"):
			out.WriteString(console.Blue("  > " + header))
		case types.ChangeAction("Remove"):
			out.WriteString(console.Red("  - " + header))
		}
		out.WriteString("\n")

		out.WriteString(ui.Indent("  ", body))
		out.WriteString("\n")
	}

	return strings.TrimSpace(out.String())
}

func GetParameters(template cft.Template, combinedParameters map[string]string, old []types.Parameter, stackExists bool) []types.Parameter {

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
		for k := range combinedParameters {
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
			if cliParam, ok := combinedParameters[k]; ok {
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

func PackageTemplate(fn string, yes bool) cft.Template {
	// Call RainBucket for side-effects in case we want to force bucket creation
	s3.RainBucket(yes)

	t, err := pkg.File(fn)
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

			err := cfn.DeleteStack(stackName, "")
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
	FixStackNameRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)
}
