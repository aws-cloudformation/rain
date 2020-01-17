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

	cmd.Execute()
	// Output:
	// Creates or updates a CloudFormation stack named <stack> from the template file <template>.
	// If you don't specify a stack name, rain will use the template filename minus its extension.
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
