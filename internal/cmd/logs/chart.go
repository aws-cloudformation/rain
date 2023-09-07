package logs

import (
	"fmt"
	"sort"
	"strings"

	_ "embed"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
)

//go:embed chart-template.html
var template string

// Create a type to wrap StackEvent so we can sort it
type evt []types.StackEvent

func (e evt) Len() int {
	return len(e)
}

func (e evt) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e evt) Less(i, j int) bool {
	return ptr.ToTime(e[i].Timestamp).Unix() > ptr.ToTime(e[j].Timestamp).Unix()
}

// createChart outputs an html file to stdout with a gantt chart
// that shows the durations for each resource of the latest stack action
func createChart(stackName string) error {

	var logs evt
	var err error

	// Get logs
	logs, err = cfn.GetStackEvents(stackName)
	if err != nil {
		return err
	}

	// Sort by timestamp
	sort.Sort(logs)

	var sb strings.Builder

	sb.WriteString("[\n")

	for _, log := range logs {
		config.Debugf("%v", log)
		iso8601, err := log.Timestamp.MarshalText()
		if err != nil {
			config.Debugf("Cannot convert timestamp: %v", log.Timestamp.String())
			continue
		}
		sb.WriteString(fmt.Sprintf("{'Id': '%v', ", *log.LogicalResourceId))
		sb.WriteString(fmt.Sprintf("'Type': '%v', ", *log.ResourceType))
		sb.WriteString(fmt.Sprintf("'Timestamp': '%v', ", string(iso8601)))
		sb.WriteString(fmt.Sprintf("'Status': '%v'}, \n", log.ResourceStatus))
	}

	sb.WriteString("]")

	data := sb.String()

	rendered := strings.Replace(template, "__DATA__", data, 1)

	fmt.Println(rendered)

	return nil
}
