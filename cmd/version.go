package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:                   "version",
	Short:                 "Display the installed version of rain",
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.String())
	},
}

func init() {
	Root.AddCommand(versionCmd)
}
