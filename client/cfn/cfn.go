package cfn

import (
	"fmt"
	"runtime"

	"github.com/aws-cloudformation/rain/util"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

const NAME = "Rain"

const VERSION = "v0.1.0"

var liveStatuses = []cloudformation.StackStatus{
	"CREATE_IN_PROGRESS",
	"CREATE_FAILED",
	"CREATE_COMPLETE",
	"ROLLBACK_IN_PROGRESS",
	"ROLLBACK_FAILED",
	"ROLLBACK_COMPLETE",
	"DELETE_IN_PROGRESS",
	"DELETE_FAILED",
	"UPDATE_IN_PROGRESS",
	"UPDATE_COMPLETE_CLEANUP_IN_PROGRESS",
	"UPDATE_COMPLETE",
	"UPDATE_ROLLBACK_IN_PROGRESS",
	"UPDATE_ROLLBACK_FAILED",
	"UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS",
	"UPDATE_ROLLBACK_COMPLETE",
	"REVIEW_IN_PROGRESS",
}

var client *cloudformation.CloudFormation

func init() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		util.Die(err)
	}

	// Set the user agent
	cfg.Handlers.Build.Remove(defaults.SDKVersionUserAgentHandler)
	cfg.Handlers.Build.PushFront(aws.MakeAddToUserAgentHandler(
		NAME,
		VERSION,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	))

	client = cloudformation.New(cfg)
}

func GetStackTemplate(stackName string) string {
	req := client.GetTemplateRequest(&cloudformation.GetTemplateInput{
		StackName: &stackName,
	})

	res, err := req.Send()
	if err != nil {
		util.Die(err)
	}

	return *res.TemplateBody
}

func StackExists(stackName string) bool {
	req := client.ListStacksRequest(&cloudformation.ListStacksInput{
		StackStatusFilter: liveStatuses,
	})

	p := req.Paginate()

	for p.Next() {
		for _, s := range p.CurrentPage().StackSummaries {
			if *s.StackName == stackName {
				return true
			}
		}
	}

	if err := p.Err(); err != nil {
		util.Die(err)
	}

	return false
}

func ListStacks(fn func(cloudformation.StackSummary)) {
	req := client.ListStacksRequest(&cloudformation.ListStacksInput{
		StackStatusFilter: liveStatuses,
	})

	p := req.Paginate()
	for p.Next() {
		for _, s := range p.CurrentPage().StackSummaries {
			fn(s)
		}
	}

	if err := p.Err(); err != nil {
		util.Die(err)
	}
}

//FIXME: What happens if the stack doesn't exist?
func GetStack(stackName string) (cloudformation.Stack, error) {
	// Get the stack properties
	req := client.DescribeStacksRequest(&cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})

	res, _ := req.Send()

	if res == nil || len(res.Stacks) != 1 {
		return cloudformation.Stack{}, fmt.Errorf("No such stack: " + stackName)
	}

	return res.Stacks[0], nil
}

func GetStackResources(stackName string) []cloudformation.StackResource {
	// Get the stack resources
	req := client.DescribeStackResourcesRequest(&cloudformation.DescribeStackResourcesInput{
		StackName: &stackName,
	})

	res, err := req.Send()
	if err != nil {
		util.Die(err)
	}

	return res.StackResources
}
