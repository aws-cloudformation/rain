package module

import (
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/spf13/cobra"
)

func checkExperimental() {
	if !experimental {
		panic("Add the --experimental argument to acknolwedge that this is an experimental feature that may change in minor version releases")
	}
}

var Cmd = &cobra.Command{
	Use:   "module <command> ",
	Short: "Interact with Rain modules in CodeArtifact",
	Long: `The rain module command can be used to publish modules to CodeArtifact, and to install modules from CodeArtifact.

	You must pass the --experimental (-x) flag to use this command, to acknowledge that it is experimental and likely to be unstable!
`,
}

var domain string
var repo string
var path string
var experimental bool

func addCommonParams(c *cobra.Command) {
	c.Flags().StringVarP(&config.Profile, "profile", "p", "", "AWS profile name; read from the AWS CLI configuration file")
	c.Flags().StringVarP(&config.Region, "region", "r", "", "AWS region to use")
	c.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
	c.Flags().BoolVarP(&experimental, "experimental", "x", false, "Acknowledge that this is an experimental feature")
	c.Flags().StringVar(&domain, "domain", "cloudformation", "The CodeArtifact domain")
	c.Flags().StringVar(&repo, "repo", "rain", "The CodeArtifact repository")
	c.Flags().StringVar(&path, "path", ".", "The local path for module files, defaults to the current directory")
}

func init() {
	Cmd.AddCommand(ModulePublishCmd)
	Cmd.AddCommand(ModuleInstallCmd)
}
