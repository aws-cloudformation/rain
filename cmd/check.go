package cmd

import (
	"fmt"
	"os"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws-cloudformation/rain/client/sts"
	"github.com/aws-cloudformation/rain/config"
	"github.com/aws-cloudformation/rain/console/spinner"
	"github.com/aws-cloudformation/rain/console/text"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:                   "check",
	Short:                 "Show your current configuration",
	Long:                  "Take a rain check.\n\nDisplay the AWS account and region that you're configured to use.\n\nAnd do nothing else for now :)",
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		spinner.Status("Getting identity...")
		id, err := sts.GetCallerId()
		if err != nil {
			panic(fmt.Errorf("Unable to load identity: %s", err))
		}
		spinner.Stop()

		fmt.Println("Account: ", text.Yellow(*id.Account))
		fmt.Println("Region:  ", text.Yellow(client.Config().Region))
		fmt.Println("Identity:", text.Yellow(*id.Arn))

		if config.Profile != "" {
			fmt.Println("Profile: ", text.Yellow(config.Profile))
		} else if profile, ok := os.LookupEnv("AWS_PROFILE"); ok {
			fmt.Println("Profile: ", text.Yellow(profile))
		}
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
