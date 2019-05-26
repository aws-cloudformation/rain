package sts

import (
	"errors"
	"runtime"

	"github.com/aws-cloudformation/rain/util"
	"github.com/aws-cloudformation/rain/version"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var client *sts.STS

func init() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		util.Die(err)
	}

	// Set the user agent
	cfg.Handlers.Build.Remove(defaults.SDKVersionUserAgentHandler)
	cfg.Handlers.Build.PushFront(aws.MakeAddToUserAgentHandler(
		version.NAME,
		version.VERSION,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	))

	client = sts.New(cfg)
}

func GetAccountId() string {
	req := client.GetCallerIdentityRequest(nil)

	res, err := req.Send()
	if err != nil {
		util.Die(errors.New("Could not get caller identity"))
	}

	return *res.Account
}
