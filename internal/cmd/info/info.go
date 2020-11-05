package info

import (
	"context"
	"fmt"
	"os"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/sts"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"

	"github.com/spf13/cobra"
)

var checkCreds = false

// Cmd is the info command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "info",
	Short:                 "Show your current configuration",
	Long:                  "Display the AWS account and region that you're configured to use.",
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		spinner.Push("Getting identity")
		id, err := sts.GetCallerID()
		if err != nil {
			panic(ui.Errorf(err, "unable to load identity"))
		}
		spinner.Pop()

		fmt.Println("Account: ", console.Yellow(*id.Account))
		fmt.Println("Region:  ", console.Yellow(aws.Config().Region))
		fmt.Println("Identity:", console.Yellow(*id.Arn))

		if config.Profile != "" {
			fmt.Println("Profile: ", console.Yellow(config.Profile))
		} else if profile, ok := os.LookupEnv("AWS_PROFILE"); ok {
			fmt.Println("Profile: ", console.Yellow(profile))
		}

		if checkCreds {
			fmt.Println()
			c, err := aws.Config().Credentials.Retrieve(context.Background())
			if err == nil {
				fmt.Println("Credentials:")
				fmt.Println("  Source:         ", console.Yellow(c.Source))
				fmt.Println("  AccessKeyId:    ", console.Yellow(c.AccessKeyID))
				fmt.Println("  SecretAccessKey:", console.Yellow(c.SecretAccessKey))
				if c.SessionToken != "" {
					fmt.Println("  SessionToken:   ", console.Yellow(c.SessionToken))
				}
				if !c.Expires.IsZero() {
					fmt.Println("  Expires:        ", console.Yellow(fmt.Sprint(c.Expires)))
				}
			}
		}
	},
}

func init() {
	Cmd.Flags().BoolVarP(&checkCreds, "creds", "c", false, "Include current AWS credentials")
}
