package sts

import (
	"github.com/aws-cloudformation/rain/client"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var stsClient *sts.STS

func getClient() *sts.STS {
	if stsClient == nil {
		stsClient = sts.New(client.GetConfig())
	}

	return stsClient
}

func GetAccountId() (string, client.Error) {
	req := getClient().GetCallerIdentityRequest(nil)

	res, err := req.Send()
	if err != nil {
		return "", client.NewError(err)
	}

	return *res.Account, nil
}
