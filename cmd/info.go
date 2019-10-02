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

var checkCreds = false

var infoCmd = &cobra.Command{
	Use:                   "info",
	Short:                 "Show your current configuration",
	Long:                  "Display the AWS account and region that you're configured to use.",
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

		if checkCreds {
			fmt.Println()
			c, err := client.Config().Credentials.Retrieve()
			if err == nil {
				fmt.Println("Credentials:")
				fmt.Println("  Source:         ", text.Yellow(c.Source))
				fmt.Println("  AccessKeyId:    ", text.Yellow(c.AccessKeyID))
				fmt.Println("  SecretAccessKey:", text.Yellow(c.SecretAccessKey))
				if c.SessionToken != "" {
					fmt.Println("  SessionToken:   ", text.Yellow(c.SessionToken))
				}
				if !c.Expires.IsZero() {
					fmt.Println("  Expires:        ", text.Yellow(fmt.Sprint(c.Expires)))
				}
			}
		}
	},
}

func init() {
	infoCmd.Flags().BoolVarP(&checkCreds, "creds", "c", false, "Include current AWS credentials")
	Root.AddCommand(infoCmd)
}
