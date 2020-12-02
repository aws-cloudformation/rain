package console

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/aws-cloudformation/rain/internal/aws/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
)

var printOnly = false
var service = "cloudformation"

// Cmd is the console command's entrypoint
var Cmd = &cobra.Command{
	Use:   "console [stack]",
	Short: "Login to the AWS console",
	Long: `Use your current credentials to create a sign-in URL for the AWS console and open it in a web browser.

If you supply a stack name (and didn't use the --service option), the browser will open with that stack selected.

The console command is only valid with an IAM role; not an IAM user.`,
	Args:                  cobra.MaximumNArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		stackName := ""
		if len(args) == 1 {
			stackName = args[0]
		}

		spinner.Push("Generating sign-in URL")
		uri, err := console.GetURI(service, stackName)
		if err != nil {
			panic(err)
		}
		spinner.Pop()

		if !printOnly {
			switch runtime.GOOS {
			case "linux":
				err = exec.Command("xdg-open", uri).Start()
			case "windows":
				err = exec.Command("rundll32", "url.dll,FileProtocolHandler", uri).Start()
			case "darwin":
				err = exec.Command("open", uri).Start()
			}
		}

		if printOnly || err != nil {
			fmt.Printf("Open the following URL in your browser: %s\n", uri)
		}
	},
}

func init() {
	Cmd.Flags().BoolVarP(&printOnly, "url", "u", false, "Just construct the sign-in URL; don't attempt to open it")
	Cmd.Flags().StringVarP(&service, "service", "s", "cloudformation", "Choose an AWS service home page to launch")
}
