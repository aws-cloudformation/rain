package stackset

import (
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/spf13/cobra"
)

const stackSetGroup = "StackSet commands"

func addCommand(label string, profileOptions bool, c *cobra.Command) {
	if label != "" {
		c.Annotations = map[string]string{"Group": label}
	}

	if profileOptions {
		c.Flags().StringVarP(&config.Profile, "profile", "p", "", "AWS profile name; read from the AWS CLI configuration file")
		c.Flags().StringVarP(&config.Region, "region", "r", "", "AWS region to use")
	}

	StackSetCmd.AddCommand(c)
}

var StackSetCmd = &cobra.Command{
	Use:   "stackset <stack_set command>",
	Short: "List CloudFormation stack sets in a given region",
	Long:  "List CloudFormation stack sets in a given region",
}

func init() {
	addCommand(stackSetGroup, true, LsCmd)
}
