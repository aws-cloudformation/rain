package bootstrap

import (
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/spf13/cobra"
)

var force = false

// Cmd is the bootstrap command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "bootstrap",
	Short:                 "Creates the artifacts bucket",
	Long:                  `Creates an S3 Bucket to hold all the artifacts generated and referenced by rain cli.`,
	Args:                  cobra.MaximumNArgs(0),
	Aliases:               []string{"bootstrap"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		s3.RainBucket(force)
	},
}

func init() {
	Cmd.Flags().BoolVarP(&force, "yes", "y", false, "creates the bucket in the account without any user confirmation")
}
