package deploy_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/deploy"
)

func Example_deploy_help() {
	os.Args = []string{
		os.Args[0],
		"--help",
	}

	deploy.Cmd.Execute()
	// Output:
	// Creates or updates a CloudFormation stack named <stack> from the template file <template>.
	// If you don't specify a stack name, rain will use the template filename minus its extension.
	//
	// If a template needs to be packaged before it can be deployed, rain will package the template first.
	// Rain will attempt to create an S3 bucket to store artifacts that it packages and deploys.
	// The bucket's name will be of the format rain-artifacts-<AWS account id>-<AWS region>.
	//
	// The config flag can be used to programmatically set tags and parameters.
	// The format is similar to the "Template configuration file" for AWS CodePipeline just without the
	// 'StackPolicy' key. The file can be in YAML or JSON format.
	//
	// JSON:
	// {
	//   "Parameters" : {
	//     "NameOfTemplateParameter" : "ValueOfParameter",
	//     ...
	//   },
	//   "Tags" : {
	//     "TagKey" : "TagValue",
	//     ...
	//   }
	// }
	//
	// YAML:
	// Parameters:
	//   NameOfTemplateParameter: ValueOfParameter
	//   ...
	// Tags:
	//   TagKey: TagValue
	//   ...
	//
	// Usage:
	//   deploy <template> [stack]
	//
	// Flags:
	//   -c, --config string            YAML or JSON file to set tags and parameters.
	//   -d, --detach                   Once deployment has started, don't wait around for it to finish.
	//   -h, --help                     help for deploy
	//   -k, --keep                     Keep deployed resources after a failure by disabling rollbacks.
	//       --params strings           Set parameter values. Use the format key1=value1,key2=value2.
	//       --role-arn string          The ARN of IAM role that AWS CloudFormation assumes to deploy the stack.
	//       --tags strings             Add tags to the stack. Use the format key1=value1,key2=value2.
	//   -t, --termination-protection   Enable  termination protection on the stack.
	//   -y, --yes                      Don't ask questions; just deploy.
}
