package bootstrap_test

import (
	"os"

	"github.com/aws-cloudformation/rain/internal/cmd/bootstrap"
)

func Example_bootstrap_help() {
	os.Args = []string{
		os.Args[0],
		"--help",
	}

	bootstrap.Cmd.Execute()
	// Creates a s3 bucket to hold all the artifacts generated and referenced by rain cli

	// Usage:
	//   rain bootstrap

	// Aliases:
	//   bootstrap, bootstrap

	// Flags:
	//       --debug            Output debugging information
	//   -h, --help             help for bootstrap
	//       --no-colour        Disable colour output
	//   -p, --profile string   AWS profile name; read from the AWS CLI configuration file
	//   -r, --region string    AWS region to use
	//   -y, --yes              creates the bucket in the account without any user confirmation
}
