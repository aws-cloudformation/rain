package ls

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/appscode/jsonpatch"
	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
)

// valueForPath finds the value for the given path and returns it as a string
func valueForPath(path string, j map[string]any) string {
	tokens := strings.Split(path, "/")
	for i, token := range tokens {
		if token == "" {
			continue
		}
		if i == len(tokens)-1 {
			return fmt.Sprintf("%v", j[token])
		} else {
			v := j[token]
			if v == nil {
				config.Debugf("unexpected valueForPath %s, %s is nil?", path, token)
				config.Debugf("j: %v", j)
				return "??"
			}
			j, ok := j[token].(map[string]any)
			if !ok {
				config.Debugf("Unexpected type for j[token]: %v", j[token])
				return "??"
			}
		}
	}
	return "?"
}

func showChangeset(stackName, changeSetName string) {
	spinner.Push("Fetching changeset details")
	cs, err := cfn.GetChangeSet(stackName, changeSetName)
	if err != nil {
		panic(ui.Errorf(err, "failed to get changeset '%s'", changeSetName))
	}
	config.Debugf("ChangeSet response: %+v", cs)
	if cs == nil {
		return
	}
	out := ""
	out += fmt.Sprintf("Arn: %v\n", *cs.ChangeSetId)
	out += fmt.Sprintf("Created: %v\n", cs.CreationTime)
	descr := ""
	if cs.Description != nil {
		descr = *cs.Description
	}
	out += fmt.Sprintf("Description: %v\n", descr)
	reason := ""
	if cs.StatusReason != nil {
		reason = "(" + *cs.StatusReason + ")"
	}
	out += fmt.Sprintf("Status: %v/%v %v\n",
		ui.ColouriseStatus(string(cs.ExecutionStatus)),
		ui.ColouriseStatus(string(cs.Status)),
		reason)
	out += "Parameters: "
	if len(cs.Parameters) == 0 {
		out += "(None)\n"
	} else {
		out += "\n"
	}
	for _, p := range cs.Parameters {
		k, v := "", ""
		if p.ParameterKey != nil {
			k = *p.ParameterKey
		}
		if p.ParameterValue != nil {
			v = *p.ParameterValue
		}
		out += fmt.Sprintf("  %s: %s\n", k, v)
	}
	out += "Changes: \n"
	for _, csch := range cs.Changes {
		if csch.ResourceChange == nil {
			continue
		}
		change := csch.ResourceChange
		rid := ""
		if change.LogicalResourceId != nil {
			rid = *change.LogicalResourceId
		}
		rt := ""
		if change.ResourceType != nil {
			rt = *change.ResourceType
		}
		pid := ""
		if change.PhysicalResourceId != nil {
			pid = *change.PhysicalResourceId
		}
		replace := ""
		switch string(change.Replacement) {
		case "True":
			replace = " [Replace]"
		case "Conditional":
			replace = " [Might replace]"
		}
		if change.Action == "Add" {
			replace = ""
		}
		changeMsg := fmt.Sprintf("  %s%s: %s (%s) %s\n",
			string(change.Action),
			replace,
			rid,
			rt,
			pid)

		switch change.Action {
		case "Add":
			out += console.Green(changeMsg)
		case "Modify":
			out += console.Blue(changeMsg)
		case "Remove":
			out += console.Red(changeMsg)
		default:
			out += changeMsg
		}

		for _, detail := range change.Details {
			config.Debugf("Detail: %+v", detail)
		}

		// Compare properties to see what has changed
		config.Debugf("Change: %+v", change)
		var before, after string
		if change.BeforeContext != nil {
			config.Debugf("Before: %v", *change.BeforeContext)
			before = *change.BeforeContext
		}
		if change.AfterContext != nil {
			config.Debugf("After: %v", *change.AfterContext)
			after = *change.AfterContext
		}
		if before != "" && after != "" {
			var beforeJson, afterJson map[string]any
			if err := json.Unmarshal([]byte(before), &beforeJson); err != nil {
				config.Debugf("%v", err)
			} else {
				if err = json.Unmarshal([]byte(after), &afterJson); err != nil {
					config.Debugf("%v", err)
				} else {
					diff := diff.CompareMaps(beforeJson, afterJson)
					config.Debugf("%s", diff.Format(true))

					// jsonpatch is a little easier to work with than Diff
					ops, err := jsonpatch.CreatePatch([]byte(before), []byte(after))
					if err != nil {
						config.Debugf("%v", err)
					} else {
						for _, op := range ops {
							path := strings.Replace(op.Path, "/", ".", -1)
							path = strings.Replace(path, ".", "", 1) // 1st instance of .
							out += console.Blue(fmt.Sprintf("    %s\n", path))
							was := valueForPath(op.Path, beforeJson)
							out += console.Blue(fmt.Sprintf("      before: %s\n", was))
							out += console.Blue(fmt.Sprintf("      after:  %s\n", op.Value))
						}
					}

				}
			}
		}
	}

	spinner.Pop()

	fmt.Println(out)

}
