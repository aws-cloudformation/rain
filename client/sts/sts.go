package sts

import (
	"context"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var stsClient *sts.Client

func getClient() *sts.Client {
	if stsClient == nil {
		stsClient = sts.New(client.Config())
	}

	return stsClient
}

func GetCallerId() (sts.GetCallerIdentityOutput, client.Error) {
	req := getClient().GetCallerIdentityRequest(nil)

	res, err := req.Send(context.Background())
	if err != nil {
		return sts.GetCallerIdentityOutput{}, client.NewError(err)
	}

	return *res.GetCallerIdentityOutput, nil
}

func GetAccountId() (string, client.Error) {
	id, err := GetCallerId()
	if err != nil {
		return "", client.NewError(err)
	}

	return *id.Account, nil
}
