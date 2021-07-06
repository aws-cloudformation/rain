package console

import (
	"github.com/spf13/cobra"
)

var printOnlyFlag = false
var serviceParam = "cloudformation"
var userName = ""

// Cmd is the console command's entrypoint
var Cmd = &cobra.Command{
	Use:   "console [stack]",
	Short: "Login to the AWS console",
	Long: `Use your current credentials to create a sign-in URL for the AWS console and open it in a web browser.

If you supply a stack name (and didn't use the --service option), the browser will open with that stack selected.

The console command is only valid with an IAM role; not an IAM user.

Unless you specify the --name/-n flag, your AWS console user name will be derived from the role name.`,
	Args:                  cobra.MaximumNArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := ""
		if len(args) == 1 {
			stackName = args[0]
		}

		Open(printOnlyFlag, serviceParam, stackName, userName)
	},
}

func init() {
	Cmd.Flags().BoolVarP(&printOnlyFlag, "url", "u", false, "Just construct the sign-in URL; don't attempt to open it")
	Cmd.Flags().StringVarP(&serviceParam, "service", "s", "cloudformation", "Choose an AWS service home page to launch")
	Cmd.Flags().StringVarP(&userName, "name", "n", "", "Specify a user name to use in the AWS console")
}
