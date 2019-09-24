package cmd

import (
	"os"

	"github.com/aws-cloudformation/rain/config"
	"github.com/spf13/cobra"
)

// Root represents the base command when called without any subcommands
var Root = &cobra.Command{
	Use:  "rain",
	Long: "Rain is a development workflow tool for working with AWS CloudFormation.",
}

func init() {
	Root.PersistentFlags().BoolVarP(&config.Debug, "debug", "", false, "Output debugging information")
	Root.PersistentFlags().StringVarP(&config.Profile, "profile", "p", "", "AWS profile name; read from the AWS CLI configuration file")
	Root.PersistentFlags().StringVarP(&config.Region, "region", "r", "", "AWS region to use")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the Root.
func Execute() {
	if err := Root.Execute(); err != nil {
		os.Exit(1)
	}
}
