package module

import (
	"fmt"
	"github.com/aws-cloudformation/rain/internal/aws/codeartifact"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/spf13/cobra"
)

func bootstrapCommand(cmd *cobra.Command, args []string) {
	config.Debugf("module bootstrap domain %s, repo %s, path %s",
		domain, repo, path)

	checkExperimental()

	bootstrap()
}

// bootstrap checks to see if the domain and repo exist,
// and if not, prompts the user to create them.
func bootstrap() {

	// Check to see if the domain exists.
	exists, err := codeartifact.DomainExists(domain)
	if err != nil {
		panic(fmt.Sprintf("error checking if domain exists: %v", err))
	}

	// If not, prompt the user to create it
	if !exists {
		msg := fmt.Sprintf("Domain %s does not exist. Would you like to create it?", domain)
		if console.Confirm(true, msg) {
			err = codeartifact.CreateDomain(domain)
			if err != nil {
				panic(fmt.Sprintf("error creating domain: %v", err))
			}
		}
	}

	// Check to see if the repo exists.
	exists, err = codeartifact.RepoExists(repo, domain)
	if err != nil {
		panic(fmt.Sprintf("error checking if repository exists: %v", err))
	}

	// If not, prompt the user to create it
	if !exists {
		msg := fmt.Sprintf("Repository %s does not exist. Would you like to create it?", repo)
		if console.Confirm(true, msg) {
			err = codeartifact.CreateRepo(repo, domain)
			if err != nil {
				panic(fmt.Sprintf("error creating repository: %v", err))
			}
		}
	}
}

var BootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap the CodeArtifact domain and repository",
	Long:  ``,
	Args:  cobra.ExactArgs(0),
	Run:   bootstrapCommand,
}

func init() {
	addCommonParams(BootstrapCmd)
}
