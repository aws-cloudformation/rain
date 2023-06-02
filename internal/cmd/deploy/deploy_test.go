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
	//   {
	//     "Parameters" : {
	//       "NameOfTemplateParameter" : "ValueOfParameter",
	//       ...
	//     },
	//     "Tags" : {
	//       "TagKey" : "TagValue",
	//       ...
	//     }
	//   }
	//
	// YAML:
	//   Parameters:
	//     NameOfTemplateParameter: ValueOfParameter
	//     ...
	//   Tags:
	//     TagKey: TagValue
	//     ...
	//
	// Usage:
	//   deploy <template> [stack]
	//
	// Flags:
	//   -c, --config string            YAML or JSON file to set tags and parameters
	//   -d, --detach                   once deployment has started, don't wait around for it to finish
	//   -h, --help                     help for deploy
	//       --ignore-unknown-params    Ignore unknown parameters
	//   -k, --keep                     keep deployed resources after a failure by disabling rollbacks
	//       --params strings           set parameter values; use the format key1=value1,key2=value2
	//       --role-arn string          ARN of an IAM role that CloudFormation should assume to deploy the stack
	//       --tags strings             add tags to the stack; use the format key1=value1,key2=value2
	//   -t, --termination-protection   enable termination protection on the stack
	//   -y, --yes                      don't ask questions; just deploy
}
