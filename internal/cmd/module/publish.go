package module

import (
	"fmt"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/spf13/cobra"
)

func publish(cmd *cobra.Command, args []string) {
	fmt.Println("publish cmd...", args[0])

	config.Debugf("module install %s, domain %s, repo %s, path %s",
		args[0], domain, repo, path)

	checkExperimental()

	bootstrap()

}

var PublishCmd = &cobra.Command{
	Use:   "publish <name>",
	Short: "Publish a directory of Rain modules to CodeArtifact",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run:   publish,
}

func init() {
	addCommonParams(PublishCmd)
}
