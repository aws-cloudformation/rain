package logs

import (
	"fmt"
	"github.com/aws-cloudformation/rain/internal/ui"

	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

var uninterestingMessages = map[string]bool{
	"Resource creation Initiated": true,
	"User Initiated":              true,
}

func printLogs(logs []*types.StackEvent) {
	for _, log := range logs {
		fmt.Printf("- %s", ui.ColouriseStatus(string(log.ResourceStatus)))

		if timeOrder {
			fmt.Print(" ")
			fmt.Print(console.Yellow(*log.LogicalResourceId))
			fmt.Print(" ")
			fmt.Print(*log.ResourceType)

		}

		if longFormat && *log.PhysicalResourceId != "" {
			fmt.Print(" ")
			fmt.Print(*log.PhysicalResourceId)
		}

		if log.ResourceStatusReason != nil {
			fmt.Print(" ")
			fmt.Print(console.White(fmt.Sprintf("%q", *log.ResourceStatusReason)))
		}

		if longFormat {
			fmt.Print(" ")
			fmt.Print(*log.Timestamp)
		}

		fmt.Println()
	}
}
