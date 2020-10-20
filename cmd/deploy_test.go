package cmd_test

import (
	"os"

	"github.com/aws-cloudformation/rain/cmd"
)

func Example_deploy_help() {
	os.Args = []string{
		os.Args[0],
		"help",
		"deploy",
	}

	cmd.Execute(cmd.Rain)
	// Output:
	// Creates or updates a CloudFormation stack named <stack> from the template file <template>.
	// If you don't specify a stack name, rain will use the template filename minus its extension.
	//
	// If a template needs to be packaged before it can be deployed, rain will package the template first.
	// Rain will attempt to create an S3 bucket to store artifacts that it packages and deploys.
	// The bucket's name will be of the format rain-artifacts-<AWS account id>-<AWS region>
	//
	// Usage:
	//   rain deploy <template> [stack]
	//
	// Flags:
	//   -d, --detach           Once deployment has started, don't wait around for it to finish.
	//   -f, --force            Don't ask questions; just deploy.
	//   -h, --help             help for deploy
	//       --params strings   Set parameter values. Use the format key1=value1,key2=value2.
	//       --tags strings     Add tags to the stack. Use the format key1=value1,key2=value2.
	//
	// Global Flags:
	//       --debug            Output debugging information
	//   -p, --profile string   AWS profile name; read from the AWS CLI configuration file
	//   -r, --region string    AWS region to use
}
