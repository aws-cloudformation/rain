package build

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws/bedrock"
	"github.com/aws-cloudformation/rain/internal/config"
)

// prompt invokes bedrock to produce a template based on the prompt
func runPrompt(p string) {
	mid := modelId(model)
	switch promptLanguage {
	case LANG_GUARD:
		promptGuard(p, mid)
	case LANG_REGO:
		promptRego(p, mid)
	default:
		promptCfn(p, mid)
	}
}

func modelId(m string) string {
	id, ok := models[m]
	if ok {
		return id
	}
	return m
}

func promptGuard(p string, mid string) {
	if isClaude2() {
		prompt := fmt.Sprintf("Write an AWS CloudFormation guard policy file that does the following:\n\n%s\n\nDo not include any explanation.\n\nWrite only the content of the cfn-guard file.\n\nOutput a valid guard rule within <guard></guard> tags.", p)
		config.Debugf("About to invoke bedrock Claude2 %s with prompt: %s", mid, prompt)
		r, err := bedrock.Invoke(prompt)
		if err != nil {
			panic(err)
		}

		// Clean up the output
		r = strings.ReplaceAll(r, "<guard>\n", "")
		r = strings.ReplaceAll(r, "</guard>", "")

		fmt.Println(r)
	} else {

		sample := `let s3_buckets_server_side_encryption = Resources.*[ Type == 'AWS::S3::Bucket'
  Metadata.cfn_nag.rules_to_suppress not exists or 
  Metadata.cfn_nag.rules_to_suppress.*.id != "W41"
  Metadata.guard.SuppressedRules not exists or
  Metadata.guard.SuppressedRules.* != "S3_BUCKET_SERVER_SIDE_ENCRYPTION_ENABLED"
]

rule S3_BUCKET_SERVER_SIDE_ENCRYPTION_ENABLED when %s3_buckets_server_side_encryption !empty {
  %s3_buckets_server_side_encryption.Properties.BucketEncryption exists
  %s3_buckets_server_side_encryption.Properties.BucketEncryption.ServerSideEncryptionConfiguration[*].ServerSideEncryptionByDefault.SSEAlgorithm in ["aws:kms","AES256"]
  <<
    Violation: S3 Bucket must enable server-side encryption.
    Fix: Set the S3 Bucket property BucketEncryption.ServerSideEncryptionConfiguration.ServerSideEncryptionByDefault.SSEAlgorithm to either "aws:kms" or "AES256"
  >>
}`

		system := fmt.Sprintf("AWS CloudFormation Guard (cfn-guard) files have a .guard file type.\n\n Write an AWS CloudFormation Guard  policy file that implements the user's request:\n\nDo not include any explanation.\n\nWrite only the content of the cfn-guard file.\n\nOutput a valid guard rule within <guard></guard> tags. The following is an example of a guard rule:\n\n%s", sample)
		config.Debugf("About to invoke bedrock %s with system: %s, prompt: %s", mid, system, p)
		r, err := bedrock.InvokeClaude3(p, mid, system)
		if err != nil {
			panic(err)
		}

		// Clean up the output
		r = strings.ReplaceAll(r, "<guard>\n", "")
		r = strings.ReplaceAll(r, "</guard>", "")

		fmt.Println(r)

	}
}

func promptRego(p string, mid string) {
	if isClaude2() {
		prompt := fmt.Sprintf("Write an Open Policy Agent (OPA) Rego (.rego) policy file that does the following:\n\n%s\n\nDo not include any explanation.\n\nWrite only the content of the rego file.\n\nOutput a valid OPA rule within <rego></rego> tags.", p)
		config.Debugf("About to invoke bedrock Claude2 %s with prompt: %s", mid, prompt)
		r, err := bedrock.Invoke(prompt)
		if err != nil {
			panic(err)
		}

		// Clean up the output
		r = strings.ReplaceAll(r, "<rego>\n", "")
		r = strings.ReplaceAll(r, "</rego>", "")

		fmt.Println(r)
	} else {

		sample := `
import input 

# deny if it creates more than 10 EC2 instances
deny_too_many_ec2[deny] {                             
    instances := [res | res:=input.Resources[_]; res.Type == "AWS::EC2::Instance"]   
    count(instances) > 10  
    deny := true
}

# deny if ssh is enabled
deny_ssh_enabled[deny] {                             
    input.Resources[_].Properties.SecurityGroupIngress[_].ToPort == 22
    deny := true
}

# deny if it creates IAM role
deny_role_created[deny] {                             
    input.Resources[_].Type == "AWS::IAM::Role"
    deny := true
}
`
		system := fmt.Sprintf("Open Policy Agent (OPA) policy files are written in Rego and have a .rego file type.\n\n Write an Open Policy Agent Rego policy file that implements the user's request:\n\nDo not include any explanation.\n\nWrite only the content of the rego file.\n\nOutput a valid rego policy within <rego></rego> tags. The following is an example of a rego policy:\n\n%s", sample)
		config.Debugf("About to invoke bedrock %s with system: %s, prompt: %s", mid, system, p)
		r, err := bedrock.InvokeClaude3(p, mid, system)
		if err != nil {
			panic(err)
		}

		// Clean up the output
		r = strings.ReplaceAll(r, "<rego>\n", "")
		r = strings.ReplaceAll(r, "</rego>", "")

		fmt.Println(r)

	}
}

func isClaude2() bool {
	mid := modelId(model)
	return strings.HasPrefix(mid, "anthropic.claude-v2")
}

func promptCfn(p string, mid string) {

	if isClaude2() {
		prompt := fmt.Sprintf("Write an AWS CloudFormation YAML template that builds the following:\n\n%s\n\nDo not include any explanation.\n\nWrite only the content of the YAML file.\n\nOutput valid YAML within <yaml></yaml> tags.", p)
		config.Debugf("About to invoke bedrock %s with prompt: %s", mid, prompt)
		r, err := bedrock.Invoke(prompt)
		if err != nil {
			panic(err)
		}

		// Clean up the output
		r = strings.ReplaceAll(r, "<yaml>\n", "")
		r = strings.ReplaceAll(r, "</yaml>", "")
		r = strings.ReplaceAll(r, " AWSTemplateFormatVersion", "AWSTemplateFormatVersion")

		fmt.Println(r)
	} else {
		system := "Write an AWS CloudFormation YAML template that implements the user's request.\n\nDo not include any explanation.\n\nWrite only the content of the YAML file.\n\nOutput valid YAML within <yaml></yaml> tags."
		config.Debugf("About to invoke bedrock %s with system: %s, prompt: %s", mid, system, p)
		r, err := bedrock.InvokeClaude3(p, mid, system)
		if err != nil {
			panic(err)
		}

		// Clean up the output
		r = strings.ReplaceAll(r, "<yaml>\n", "")
		r = strings.ReplaceAll(r, "</yaml>", "")
		r = strings.ReplaceAll(r, " AWSTemplateFormatVersion", "AWSTemplateFormatVersion")

		fmt.Println(r)

	}
}
