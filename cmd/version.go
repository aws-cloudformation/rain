package cmd

import (
	"fmt"
	"runtime"

	"github.com/aws-cloudformation/rain/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:                   "version",
	Short:                 "Display the installed version of rain",
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s %s %s/%s\n", version.NAME, version.VERSION, runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
