package sts

import (
	"context"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func getClient() *sts.Client {
	return sts.New(client.Config())
}

// GetCallerID returns the identity of the current IAM principal
func GetCallerID() (sts.GetCallerIdentityOutput, client.Error) {
	req := getClient().GetCallerIdentityRequest(nil)

	res, err := req.Send(context.Background())
	if err != nil {
		return sts.GetCallerIdentityOutput{}, client.NewError(err)
	}

	return *res.GetCallerIdentityOutput, nil
}

// GetAccountID gets the account number of the current AWS account
func GetAccountID() (string, client.Error) {
	id, err := GetCallerID()
	if err != nil {
		return "", client.NewError(err)
	}

	return *id.Account, nil
}
