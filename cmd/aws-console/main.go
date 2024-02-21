package main

import (
	"github.com/spf13/cobra"

	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/cmd/console"
	"github.com/aws-cloudformation/rain/internal/config"
)

var printOnly = false
var logout = false
var userName = ""

// Cmd is the console command's entrypoint
var Cmd = &cobra.Command{
	Use:   "aws-console [service]",
	Short: "Login to the AWS console",
	Long: `Use your current credentials to create a sign-in URL for the AWS console and open it in a web browser.

The console command is only valid with an IAM role; not an IAM user.

Unless you specify the --name/-n flag, your AWS console user name will be derived from the role name.`,
	Args:                  cobra.MaximumNArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		service := "console"
		if len(args) == 1 {
			service = args[0]
		}

		console.Open(printOnly, logout, service, "", userName)
	},
}

func init() {
	Cmd.Flags().BoolVarP(&printOnly, "url", "u", false, "Just construct the sign-in URL; don't attempt to open it")
	Cmd.Flags().BoolVarP(&logout, "logout", "l", false, "Log out of the AWS console")
	Cmd.Flags().StringVarP(&config.Profile, "profile", "p", "", "AWS profile name; read from the AWS CLI configuration file")
	Cmd.Flags().StringVarP(&config.Region, "region", "r", "", "AWS region to use")
	Cmd.Flags().StringVarP(&userName, "name", "n", "", "Specify a user name to use in the AWS console")
}

func main() {
	cmd.Wrap("aws-console", Cmd)
}
