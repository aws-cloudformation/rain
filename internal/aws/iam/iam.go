package iam

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	aws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/s11n"
	awsgo "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"gopkg.in/yaml.v3"
)

func getClient() *iam.Client {
	return iam.NewFromConfig(aws.Config())
}

// Get the role arn of the caller based on the aws config
func GetCallerArn(config awsgo.Config) (string, error) {
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

// Simulate actions on a resource.
// The role arg is optional, if not provided, the current aws config will be used.
func Simulate(
	actions []string,
	resource string,
	roleArn string,
	spinnerCallback func(string)) (bool, []string) {

	awsConfig := aws.Config()
	client := iam.NewFromConfig(awsConfig)
	input := &iam.SimulatePrincipalPolicyInput{}
	input.ResourceArns = []string{resource}

	messages := make([]string, 0)

	input.PolicySourceArn = &roleArn

	// Return value
	allowed := true

	// We have to check these one at a time since we can't easily predict
	// which of the actions we get from the type description schema have
	// different authorization types
	for _, action := range actions {
		input.ActionNames = []string{action}

		spinnerCallback(action)

		res, err := client.SimulatePrincipalPolicy(context.Background(), input)

		spinner.Pop()

		if err != nil {
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
			messages = append(messages, err.Error())
			return false, messages
		}
		for _, evalResult := range res.EvaluationResults {
			if evalResult.EvalDecision != types.PolicyEvaluationDecisionTypeAllowed {
				messages = append(messages, fmt.Sprintf("%v not allowed on %v", *evalResult.EvalActionName, *evalResult.EvalResourceName))
				allowed = false
			}
		}
	}
	return allowed, messages
}

// Check to see if a role exists in the account
func RoleExists(roleArn string) bool {
	tokens := strings.Split(roleArn, ":role/")
	lastToken := tokens[len(tokens)-1]
	config.Debugf("RoleExists %v %v", roleArn, lastToken)
	res, err := getClient().GetRole(context.Background(), &iam.GetRoleInput{
		RoleName: &lastToken,
	})
	if err != nil {
		config.Debugf("RoleExists GetRole Error for %v: %v", roleArn, err)
		return false
	}
	config.Debugf("RoleExists found %v: %v", roleArn, res)
	return true
}

// Check to see if the principal exists in the account
func PrincipalExists(principal string) (bool, error) {

	config.Debugf("PrincipalExists %v", principal)

	if principal == "*" {
		return true, nil
	}

	// TODO - need to check if it's created in this template and hasn't been deployed

	// What kind of principal is it?
	// Is there a way to simply ask the API "does a resource with this arn exist?"
	// If not then we need to figure out what type of resource it is and ask the service

	// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_principal.html

	// Regex to see if it's a number, arn, etc

	var accountIdRegex = regexp.MustCompile(`^\d{12}$`)
	if accountIdRegex.MatchString(principal) {
		config.Debugf("PrincipalExists %v is an account id", principal)
		// Assume that the account exists
		return true, nil
	}

	var rootRegex = regexp.MustCompile(`arn:aws:iam::\d{12}:root`)
	if rootRegex.MatchString(principal) {
		config.Debugf("PrincipalExists %v is an account root", principal)
		// Assume that the account exists
		return true, nil
	}

	var roleRegex = regexp.MustCompile(`arn:aws:iam::\d{12}:role/[a-zA-Z0-9_@=\\-]+`)
	if roleRegex.MatchString(principal) {
		config.Debugf("PrincipalExists %v is a role", principal)
		if RoleExists(principal) {
			return true, nil
		}
	}

	return false, nil
}

// Check a PolicyDocument to make sure it will not result in failures
func CheckPolicyDocument(doc *yaml.Node) (bool, error) {

	policyOk := true

	_, statements := s11n.GetMapValue(doc, "Statement")
	if statements != nil {
		for _, statement := range statements.Content {
			_, principals := s11n.GetMapValue(statement, "Principal")
			if principals != nil {
				for i, principal := range principals.Content {
					if i%2 == 0 && principal.Value == "AWS" {
						for _, p := range principals.Content[i+1].Content {
							config.Debugf("About to check if principal exists: %v", p)
							if p.Kind == yaml.MappingNode {
								config.Debugf("principal is a map")

								// This is.. hard. We need to resolve !Sub, !Ref, etc
								// TODO
							} else {
								exists, err := PrincipalExists(p.Value)
								if err != nil || !exists {
									config.Debugf("Principal not found: %v", principal)
									policyOk = false
								}
							}
						}
					}
				}
			}
		}
	}

	return policyOk, nil
}
