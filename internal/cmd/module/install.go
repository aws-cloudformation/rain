package module

import (
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/spf13/cobra"
)

func install(cmd *cobra.Command, args []string) {
	config.Debugf("module install %s, domain %s, repo %s, path %s",
		args[0], domain, repo, path)

	checkExperimental()

	bootstrap()
}

var InstallCmd = &cobra.Command{
	Use:   "install <name>",
	Short: "Install a package of Rain modules from CodeArtifact",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run:   install,
}

func init() {
	addCommonParams(InstallCmd)
}
