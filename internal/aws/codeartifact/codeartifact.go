// Package codeartifact provides functions that interact with the AWS CodeArtifact API
package codeartifact

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codeartifact/types"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	rainaws "github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/aws-sdk-go-v2/service/codeartifact"
	"github.com/aws/smithy-go"
)

func getClient() *codeartifact.Client {
	return codeartifact.NewFromConfig(rainaws.Config())
}

// DomainExists checks if a domain exists
func DomainExists(name string) (bool, error) {
	client := getClient()
	res, err := client.DescribeDomain(context.Background(),
		&codeartifact.DescribeDomainInput{Domain: &name})
	if err != nil {
		// Check to see if this is a ResourceNotFoundException
		var ae smithy.APIError
		if errors.As(err, &ae) {
			config.Debugf("code: %s, message: %s, fault: %s", ae.ErrorCode(), ae.ErrorMessage(), ae.ErrorFault().String())

			if ae.ErrorCode() == "ResourceNotFoundException" {
				return false, nil
			}
		}
		return false, err
	}
	return res.Domain != nil, nil
}

// CreateDomain creates a domain
func CreateDomain(name string) error {
	client := getClient()
	_, err := client.CreateDomain(context.Background(),
		&codeartifact.CreateDomainInput{Domain: &name})
	return err
}

// RepoExists checks if a repo exists
func RepoExists(name string, domain string) (bool, error) {
	client := getClient()
	res, err := client.DescribeRepository(context.Background(),
		&codeartifact.DescribeRepositoryInput{Domain: &domain, Repository: &name})
	if err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) {
			config.Debugf("code: %s, message: %s, fault: %s", ae.ErrorCode(), ae.ErrorMessage(), ae.ErrorFault().String())

			if ae.ErrorCode() == "ResourceNotFoundException" {
				return false, nil
			}
		}
		return false, err
	}
	return res.Repository != nil, nil
}

// CreateRepo creates a repo
func CreateRepo(name string, domain string) error {
	client := getClient()
	_, err := client.CreateRepository(context.Background(),
		&codeartifact.CreateRepositoryInput{Domain: &domain, Repository: &name})
	return err
}

type PackageInfo struct {
	Domain        string
	Repo          string
	Name          string
	Version       string
	DirectoryPath string
}

func ZipDirectory(directoryPath string) ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		config.Debugf("Walking path: %s", path)

		if info.IsDir() {
			return nil
		}

		// Create a new file header
		relPath, err := filepath.Rel(directoryPath, path)
		if err != nil {
			return err
		}

		config.Debugf("Relative path: %s", relPath)

		fileHeader, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		fileHeader.Name = relPath

		config.Debugf("File header: %+v", fileHeader)

		// Create a new file entry in the zip archive
		fileWriter, err := zipWriter.CreateHeader(fileHeader)
		if err != nil {
			return err
		}

		// Open the file for reading
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				config.Debugf("Error closing file: %s", err)
			}
		}(file)

		// Copy the file contents to the zip entry
		_, err = io.Copy(fileWriter, file)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	err = zipWriter.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

const DefaultNamespace = "default"
const DefaultVersion = "0.1.0"

// semverIsGreater checks if a version string is greater than another version string
//
//	The version strings are in the format "major.minor.patch"
//	The function returns true if the first version string is greater than the second version string,
//	and false otherwise.
func semverIsGreater(a, b string) (bool, error) {
	// Split the version strings into their individual components
	aParts := strings.Split(a, ".")
	if len(aParts) != 3 {
		return false, fmt.Errorf("invalid version format: %s", a)
	}
	bParts := strings.Split(b, ".")
	if len(bParts) != 3 {
		return false, fmt.Errorf("invalid version format: %s", b)
	}

	// Compare the components pairwise
	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		aPart, err := strconv.Atoi(aParts[i])
		if err != nil {
			return false, err
		}
		bPart, err := strconv.Atoi(bParts[i])
		if err != nil {
			return false, err
		}
		if aPart > bPart {
			return true, nil
		} else if aPart < bPart {
			return false, nil
		}
	}

	// If all components are equal, the version strings are equal
	return false, nil
}

func incrementSemverMinorVersion(version string) (string, error) {
	// Split the version string into its components
	parts := strings.Split(version, ".")

	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version format: %s", version)
	}

	// Get the minor version component
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", err
	}

	// Increment the minor version component
	parts[1] = strconv.Itoa(minor + 1)
	parts[2] = "0"

	// Return the new version string
	return strings.Join(parts, "."), nil
}

// Publish publishes a package
func Publish(packageInfo *PackageInfo) error {

	config.Debugf("Package: %+v", packageInfo)

	client := getClient()

	// Zip the contents of the directory in memory
	zipFile, err := ZipDirectory(packageInfo.DirectoryPath)
	if err != nil {
		return fmt.Errorf("failed to zip directory: %v", err)
	}

	// Create a SHA265 hash of the zipFile file reader
	hash := sha256.New()
	zipReader := bytes.NewReader(zipFile)
	if _, err := io.Copy(hash, zipReader); err != nil {
		return err
	}
	hashValue := hash.Sum(nil)

	config.Debugf("Hash value: %x", hashValue)

	var newVersion string

	// Reset the zipFile reader
	//zipReader = bytes.NewReader(zipFile)

	// If a package version was not specified, get the current
	// version of the package from the codeartifact api,
	// and increment the build number by 1
	if packageInfo.Version == "" {
		// Call the codeartifact api to get the current version of the package
		res, err := client.ListPackageVersions(context.Background(),
			&codeartifact.ListPackageVersionsInput{
				Domain:     &packageInfo.Domain,
				Repository: &packageInfo.Repo,
				Package:    &packageInfo.Name,
				Format:     types.PackageFormatGeneric,
				Namespace:  aws.String(DefaultNamespace),
			})
		var ae smithy.APIError
		if err != nil && errors.As(err, &ae) {
			config.Debugf("code: %s, message: %s, fault: %s", ae.ErrorCode(), ae.ErrorMessage(), ae.ErrorFault().String())

			if ae.ErrorCode() == "ResourceNotFoundException" {
				// If the package does not exist, create it
				newVersion = DefaultVersion
			}
		} else if err != nil {
			return err
		}

		if newVersion == "" {
			// Get the current version of the package
			config.Debugf("Package versions: %+v", res.Versions)

			// Iterate through the versions and find the latest version
			var latestVersion string
			for _, version := range res.Versions {
				if version.Version == nil {
					continue // Shouldn't happen
				}
				isGreater, err := semverIsGreater(*version.Version, latestVersion)
				if err != nil {
					return fmt.Errorf("unable to compare versions %s and %s: %v", *version.Version, latestVersion, err)
				}
				if isGreater {
					latestVersion = *version.Version
				}
			}

			config.Debugf("Latest version: %s", latestVersion)

			// Increment the minor version number by 1
			// For example, version 1.2.3 becomes 1.3.0
			newVersion, err = incrementSemverMinorVersion(latestVersion)
			if err != nil {
				return fmt.Errorf("unable to increment version %s: %v", latestVersion, err)
			}
		}
	} else {
		newVersion = packageInfo.Version
	}

	config.Debugf("About to publish new version: %s", newVersion)

	// Call the codeartifact api to publish the package version

	return nil
}

//
//func Install() error {
//
//}
