package cmd

import (
	"os"

	"github.com/aws-cloudformation/rain/config"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:  "rain",
	Long: "Rain is a development workflow tool for working with AWS CloudFormation.",
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&config.Profile, "profile", "p", "", "AWS profile name; read from the AWS CLI configuration file")
	rootCmd.PersistentFlags().StringVarP(&config.Region, "region", "r", "", "AWS region to use")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
