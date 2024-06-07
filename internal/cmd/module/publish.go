package module

import (
	"fmt"
	"github.com/aws-cloudformation/rain/internal/aws/codeartifact"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/spf13/cobra"
)

func publish(cmd *cobra.Command, args []string) {
	config.Debugf("module install %s, domain %s, repo %s, path %s",
		args[0], domain, repo, path)

	checkExperimental()

	bootstrap()

	packageInfo := &codeartifact.PackageInfo{
		Name:          args[0],
		DirectoryPath: path,
		Domain:        domain,
		Repo:          repo,
		Version:       version,
	}

	err := codeartifact.Publish(packageInfo)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Successfully published %s:%s:%s@%s\n",
		domain, repo, args[0], packageInfo.Version)
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
	// Add a version param
	PublishCmd.Flags().StringVar(&version, "version", "", "Version of the module to publish")
}
