package module

import (
	"crypto/sha256"
	"fmt"
	"github.com/aws-cloudformation/rain/internal/aws/codeartifact"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/spf13/cobra"
	"io"
	"os"
)

func install(cmd *cobra.Command, args []string) {
	config.Debugf("module install %s, domain %s, repo %s, path %s",
		args[0], domain, repo, path)

	checkExperimental()

	bootstrap()

	packageInfo := &codeartifact.PackageInfo{
		Name:          args[0],
		Domain:        domain,
		Repo:          repo,
		DirectoryPath: path,
		Version:       version,
	}

	var version string
	var err error

	// If the version is not specified, query CodeArtifact for the latest version
	if packageInfo.Version == "" {
		version, err = codeartifact.GetLatestPackageVersion(packageInfo)
		if err != nil {
			panic(fmt.Errorf("failed to get latest package version: %v", err))
		}
	} else {
		version = packageInfo.Version
	}

	packageInfo.Version = version

	spinner.Push(fmt.Sprintf("Downloading version %s", version))

	asset, err := codeartifact.GetAsset(packageInfo)
	if err != nil {
		spinner.Pop()
		panic(err)
	}

	spinner.Pop()

	spinner.Push("Validating package integrity")

	// Save the asset content to a temp file
	pFile, err := os.CreateTemp("", packageInfo.Name+"-*.zip")
	if err != nil {
		panic(err)
	}
	defer func(pFile *os.File) {
		err := pFile.Close()
		if err != nil {
			config.Debugf("Error closing file: %s", err)
		}
	}(pFile)

	config.Debugf("Saving asset content to %s", pFile.Name())

	// Write the asset content to the temp file
	if _, err := io.Copy(pFile, asset); err != nil {
		panic(err)
	}

	// Seek to the beginning of the file
	if _, err := pFile.Seek(0, 0); err != nil {
		panic(err)
	}

	// Create a sha256 hash of the asset content and verify it
	hash := sha256.New()
	// Read the contents of the temporary pFile and generate a sha256 hash
	if _, err := io.Copy(hash, pFile); err != nil {
		panic(err)
	}
	hashValue := hash.Sum(nil)

	// Convert the hash value to a hex string
	hashString := fmt.Sprintf("%x", hashValue)

	config.Debugf("Hash value: %x", hashString)

	// Reset pFile to the beginning
	if _, err := pFile.Seek(0, 0); err != nil {
		panic(err)
	}

	existingHash, err := codeartifact.GetAssetHashForPackage(packageInfo)
	if err != nil {
		panic(err)
	}

	if existingHash != hashString {
		panic(fmt.Errorf("hash does not match: %s != %s", existingHash, hashString))
	}

	spinner.Pop()

	// Create a directory to store the package
	if packageInfo.DirectoryPath == "" {
		packageInfo.DirectoryPath = "." // Use the current directory
	}

	if packageInfo.DirectoryPath != "." {
		if console.Confirm(true,
			fmt.Sprintf("About to create %s to store modules, ok?", packageInfo.DirectoryPath)) {

			// Create the directory if it does not exist
			if _, err := os.Stat(packageInfo.DirectoryPath); os.IsNotExist(err) {
				if err := os.MkdirAll(packageInfo.DirectoryPath, 0755); err != nil {
					panic(err)
				}
			}
		} else {
			panic(fmt.Errorf("cancelled"))
		}
	}

	// Unzip pFile into the new package directory
	err = codeartifact.Unzip(pFile, packageInfo.DirectoryPath)
	if err != nil {
		panic(err)
	}
	config.Debugf("Unzipped package to %s", packageInfo.DirectoryPath)

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
	InstallCmd.Flags().StringVar(&version, "version", "", "Version of the module to install")
}
