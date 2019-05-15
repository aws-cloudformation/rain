package cmd

import (
	"runtime"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

const NAME = "Rain-golden"
const VERSION = "v0.1.0"

var cfnClient *cloudformation.CloudFormation

func init() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic("Could not load config: " + err.Error())
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

	cfnClient = cloudformation.New(cfg)
}
