package cmd_test

import (
	"os"
	"testing"

	"github.com/aws-cloudformation/rain/cmd"
	"github.com/aws-cloudformation/rain/console"
)

func TestMain(m *testing.M) {
	console.HasColour = false
	os.Exit(m.Run())
}

func Example_rainhelp() {
	os.Args = []string{
		os.Args[0],
	}

	cmd.Execute()
	// Output:
	// Rain is what happens when you have a lot of CloudFormation ;)
	//
	// Usage:
	//   rain [command]
	//
	// Stack commands:
	//   cat         Get the CloudFormation template from a running stack
	//   deploy      Deploy a CloudFormation stack from a local template
	//   logs        Show the event log for the named stack
	//   ls          List running CloudFormation stacks
	//   rm          Delete a running CloudFormation stack
	//   watch       Display an updating view of a CloudFormation stack
	//
	// Template commands:
	//   build       Create CloudFormation templates
	//   check       Validate a CloudFormation template against the spec
	//   diff        Compare CloudFormation templates
	//   fmt         Format CloudFormation templates
	//   tree        Find dependencies of Resources and Outputs in a local template
	//
	// Other Commands:
	//   help        Help about any command
	//   info        Show your current configuration
	//
	// Flags:
	//       --debug            Output debugging information
	//   -h, --help             help for rain
	//   -p, --profile string   AWS profile name; read from the AWS CLI configuration file
	//   -r, --region string    AWS region to use
	//       --version          version for rain
	//
	// Use "rain [command] --help" for more information about a command.
}
