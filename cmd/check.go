package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/client"
	"github.com/aws-cloudformation/rain/client/sts"
	"github.com/aws-cloudformation/rain/util"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:                   "check",
	Short:                 "Show your current configuration",
	Long:                  "Take a rain check.\n\nDisplay the AWS account and region that you're configured to use.\n\nAnd do nothing else for now :)",
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		id, err := sts.GetCallerId()
		if err != nil {
			util.Die(err)
		}

		cfg := client.GetConfig()

		fmt.Println("Account: ", util.Yellow(*id.Account))
		fmt.Println("Region:  ", util.Yellow(cfg.Region))
		fmt.Println("Identity:", util.Yellow(*id.Arn))

	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
