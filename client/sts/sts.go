package sts

import (
	"runtime"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws-cloudformation/rain/util"
	"github.com/aws-cloudformation/rain/version"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var stsClient *sts.STS

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

	stsClient = sts.New(cfg)
}

func GetAccountId() (string, client.Error) {
	req := stsClient.GetCallerIdentityRequest(nil)

	res, err := req.Send()
	if err != nil {
		return "", client.NewError(err)
	}

	return *res.Account, nil
}
