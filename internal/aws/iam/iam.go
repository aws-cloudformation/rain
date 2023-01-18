package iam

import (
	"context"
	"fmt"
	"strings"

	aws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
	awsgo "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// func getClient() *iam.Client {
// 	return iam.NewFromConfig(aws.Config())
// }

// Get the role arn of the caller based on the aws config
func getCallerArn(config awsgo.Config, iamClient *iam.Client, role string) (string, error) {
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
	if role == "" {
		assumedRole := strings.Split(*stsRes.Arn, "assumed-role/")[1]
		actualRoleName := strings.Split(assumedRole, "/")[0]
		return fmt.Sprintf("arn:aws:iam::%v:role/%v", accountId, actualRoleName), nil
	} else {
		return fmt.Sprintf("arn:aws:iam::%v:role/%v", accountId, role), nil
	}
}

// Simulate actions on a resource.
// The role arg is optional, if not provided, the current aws config will be used.
func Simulate(actions []string, resource string, role string) (bool, error) {
	awsConfig := aws.Config()
	client := iam.NewFromConfig(awsConfig)
	input := &iam.SimulatePrincipalPolicyInput{}
	input.ResourceArns = []string{resource}

	arn, err := getCallerArn(awsConfig, client, role)
	if err != nil {
		fmt.Println("Could not get caller arn", err)
		return false, err
	}
	config.Debugf("Caller role arn: %v", arn)
	input.PolicySourceArn = &arn

	// Return value
	allowed := true

	// We have to check these one at a time since we can't easily predict
	// which of the actions we get from the type description schema have
	// different authorization types
	for _, action := range actions {
		input.ActionNames = []string{action}

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

			/*
				Policy simulation failed operation error IAM:
				SimulatePrincipalPolicy, https response error StatusCode: 400,
				RequestID: 0f38824c-7f07-491b-a156-b5fb9fdd03fc,
				InvalidInput: Invalid Input Actions:
				[s3:CreateBucket,s3:PutBucketTagging,s3:PutAnalyticsConfiguration,s3:PutEncryptionConfiguration,s3:PutBucketCORS,s3:PutInventoryConfiguration,s3:PutLifecycleConfiguration,s3:PutMetricsConfiguration,s3:PutBucketNotification,s3:PutBucketWebsite,s3:PutAccelerateConfiguration,s3:PutBucketPublicAccessBlock,s3:PutReplicationConfiguration,s3:PutObjectAcl,s3:PutBucketObjectLockConfiguration,s3:GetBucketAcl,s3:ListBucket,iam:PassRole,s3:DeleteObject,s3:PutBucketLogging,s3:PutBucketVersioning,s3:PutBucketOwnershipControls]
				and
				[s3:PutBucketReplication,s3:PutObjectLockConfiguration,s3:PutBucketIntelligentTieringConfiguration]
				require different authorization information.
				Please refer to the documentation for more details: https://docs.aws.amazon.com/IAM/latest/APIReference/API_SimulatePrincipalPolicy.html

				(The docs don't have any more details...)
				Checking them one at a time to get around this.
			*/
			return false, err
		}
		for _, evalResult := range res.EvaluationResults {
			if evalResult.EvalDecision != types.PolicyEvaluationDecisionTypeAllowed {
				fmt.Println(*evalResult.EvalActionName, "not allowed on", *evalResult.EvalResourceName)
				allowed = false
			}
		}
	}
	return allowed, nil
}
