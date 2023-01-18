package iam

import (
	"context"
	"fmt"
	"strings"

	aws "github.com/aws-cloudformation/rain/internal/aws"
	awsgo "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// func getClient() *iam.Client {
// 	return iam.NewFromConfig(aws.Config())
// }

// Get the role arn of the caller based on the aws config
func getCallerArn(config awsgo.Config, iamClient *iam.Client) (string, error) {
	stsClient := sts.NewFromConfig(config)
	stsRes, stsErr := stsClient.GetCallerIdentity(context.Background(),
		&sts.GetCallerIdentityInput{})
	if stsErr != nil {
		fmt.Println("Unable to get caller identity", stsErr)
		return "", stsErr
	}
	// Convert this
	// arn:aws:sts::755952356119:assumed-role/Admin/ezbeard-Isengard
	// to this:
	// arn:aws:iam::755952356119:role/Admin
	//
	// Will this work consistently for other SSO providers?
	// Is there a programmatic way to retrieve the actual role?
	sts := strings.Split(*stsRes.Arn, "sts::")[1]
	accountId := strings.Split(sts, ":")[0]
	assumedRole := strings.Split(*stsRes.Arn, "assumed-role/")[1]
	actualRoleName := strings.Split(assumedRole, "/")[0]
	return fmt.Sprintf("arn:aws:iam::%v:role/%v", accountId, actualRoleName), nil
}

// Simulate an action on a resource
func Simulate(action string, resource string) (bool, error) {
	config := aws.Config()
	client := iam.NewFromConfig(config)
	input := &iam.SimulatePrincipalPolicyInput{}
	input.ActionNames = []string{action}
	input.ResourceArns = []string{resource}

	arn, err := getCallerArn(config, client)
	if err != nil {
		fmt.Println("Could not get caller arn", err)
		return false, err
	}
	// TODO: Allow user to specify a role as a command line arg
	input.PolicySourceArn = &arn

	res, err := client.SimulatePrincipalPolicy(context.Background(), input)
	if err != nil {
		fmt.Println("Policy simulation failed", err)
		/*
			Policy simulation failed operation error IAM:
			SimulatePrincipalPolicy, https response error StatusCode: 400,
			RequestID: 2d02e533-05ae-4202-acad-caeefa16757e,
			InvalidInput: Invalid Entity Arn:
			arn:aws:sts::755952356119:assumed-role/Admin/ezbeard-Isengard
			does not clearly define entity type and name.

			This is the actual role: arn:aws:iam::755952356119:role/Admin

			(Correcting for this in getCallerArn)

		*/
		return false, err
	}
	allowed := true
	for _, evalResult := range res.EvaluationResults {
		if evalResult.EvalDecision != types.PolicyEvaluationDecisionTypeAllowed {
			fmt.Println(evalResult.EvalActionName, "not allowed on", evalResult.EvalResourceName)
			allowed = false
		}
	}

	return allowed, nil
}
