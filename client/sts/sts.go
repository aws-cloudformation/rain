package sts

import (
	"context"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func getClient() *sts.Client {
	return sts.New(client.Config())
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
