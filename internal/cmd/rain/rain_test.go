package rain_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/rain"
)

func Example_rainhelp() {
	os.Args = []string{
		os.Args[0],
	}

	rain.Cmd.Execute()
	// Output:
	// Rain is a command line tool for working with AWS CloudFormation templates and stacks
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
	//   stackset    This command manipulates stack sets.
	//   watch       Display an updating view of a CloudFormation stack
	//
	// Template commands:
	//   bootstrap   Creates the artifacts bucket
	//   build       Create CloudFormation templates
	//   diff        Compare CloudFormation templates
	//   fmt         Format CloudFormation templates
	//   forecast    Predict deployment failures
	//   merge       Merge two or more CloudFormation templates
	//   pkg         Package local artifacts into a template
	//   tree        Find dependencies of Resources and Outputs in a local template
	//
	// Other Commands:
	//   completion  Generate the autocompletion script for the specified shell
	//   console     Login to the AWS console
	//   help        Help about any command
	//   info        Show your current configuration
}
