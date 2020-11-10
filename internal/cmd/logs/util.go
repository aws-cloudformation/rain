package logs

import (
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/awslabs/smithy-go/ptr"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
)

var uninterestingMessages = map[string]bool{
	"Resource creation Initiated": true,
	"User Initiated":              true,
	"Transformation succeeded":    true,
}

type events []*types.StackEvent

func (e events) Len() int {
	return len(e)
}

func (e events) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e events) Less(i, j int) bool {
	return ptr.ToTime(e[i].Timestamp).Unix() < ptr.ToTime(e[j].Timestamp).Unix()
}

func printLogs(logs events) {
	for _, log := range logs {
		fmt.Printf("%s %s/%s (%s) %s",
			console.White(ptr.ToTime(log.Timestamp).Format(time.Stamp)),
			ptr.ToString(log.StackName),
			console.Yellow(ptr.ToString(log.LogicalResourceId)),
			ptr.ToString(log.ResourceType),
			ui.ColouriseStatus(string(log.ResourceStatus)),
		)

		if log.ResourceStatusReason != nil {
			fmt.Printf(" %q", ptr.ToString(log.ResourceStatusReason))
		}

		fmt.Println()
	}
}

func getLogs(stackName, resourceName string) (events, error) {
	spinner.Push(fmt.Sprintf("Getting logs for stack '%s'", stackName))

	var logs events
	var err error

	// Get logs
	logs, err = cfn.GetStackEvents(stackName)
	if err != nil {
		return nil, err
	}

	if resourceName != "" {
		// Filter by resource
		newLogs := make([]*types.StackEvent, 0)

		for _, log := range logs {
			if ptr.ToString(log.LogicalResourceId) == resourceName {
				newLogs = append(newLogs, log)
			}
		}

		logs = newLogs
	} else {
		// See if we have nested stacks (don't get these if we've specified a resource)
		resources, err := cfn.GetStackResources(stackName)
		if err != nil {
			return nil, err
		}

		for _, resource := range resources {
			if ptr.ToString(resource.ResourceType) == "AWS::CloudFormation::Stack" {
				if resource.PhysicalResourceId != nil {
					nestedLogs, err := getLogs(ptr.ToString(resource.PhysicalResourceId), "")
					if err != nil {
						return nil, err
					}

					logs = append(logs, nestedLogs...)
				}
			}
		}
	}

	// Filter out uninteresting messages
	newLogs := make([]*types.StackEvent, 0)
	for _, log := range logs {
		if allLogs || (log.ResourceStatusReason != nil && !uninterestingMessages[*log.ResourceStatusReason]) {
			newLogs = append(newLogs, log)
		}
	}
	logs = newLogs

	// Sort by timestamp
	sort.Sort(logs)

	// Reverse order
	for i := len(logs)/2 - 1; i >= 0; i-- {
		j := len(logs) - 1 - i
		logs[i], logs[j] = logs[j], logs[i]
	}

	spinner.Pop()

	return logs, nil
}
