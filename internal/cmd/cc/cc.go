package cc

import (
	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/dc"
	"github.com/spf13/cobra"
)

// Args
var params []string
var tags []string
var configFilePath string
var Experimental bool
var yes bool
var ignoreUnknownParams bool
var unlock string

// Globals (seems bad..? but cumbersome to pass them around)
var deployedTemplate cft.Template
var resMap map[string]*Resource
var templateConfig *dc.DeployConfig

var Cmd = &cobra.Command{
	Use:   "cc <command>",
	Short: "Interact with templates using Cloud Control API instead of CloudFormation",
	Long: `You must pass the --experimental (-x) flag to use this command, to acknowledge that it is experimental and likely to be unstable!
`,
}

func addCommonParams(c *cobra.Command) {
	c.Flags().StringVarP(&config.Profile, "profile", "p", "", "AWS profile name; read from the AWS CLI configuration file")
	c.Flags().StringVarP(&config.Region, "region", "r", "", "AWS region to use")

	c.Flags().StringVar(&s3.BucketName, "s3-bucket", "", "Name of the S3 bucket that is used to upload assets")
	c.Flags().StringVar(&s3.BucketKeyPrefix, "s3-prefix", "", "Prefix to add to objects uploaded to S3 bucket")
	c.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
	c.Flags().BoolVarP(&Experimental, "experimental", "x", false, "Acknowledge that this is an experimental feature")
}

func init() {
	Cmd.AddCommand(CCDeployCmd)
	Cmd.AddCommand(CCRmCmd)
	Cmd.AddCommand(CCStateCmd)
	Cmd.AddCommand(CCDriftCmd)
}
