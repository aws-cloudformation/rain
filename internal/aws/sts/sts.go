package sts

import (
	"context"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func getClient() *sts.Client {
	return sts.NewFromConfig(aws.Config())
}

// GetCallerID returns the identity of the current IAM principal
func GetCallerID() (sts.GetCallerIdentityOutput, error) {
	res, err := getClient().GetCallerIdentity(context.Background(), nil)
	if err != nil {
		return sts.GetCallerIdentityOutput{}, err
	}

	return *res, nil
}

// GetAccountID gets the account number of the current AWS account
func GetAccountID() (string, error) {
	id, err := GetCallerID()
	if err != nil {
		return "", err
	}

	return *id.Account, nil
}
