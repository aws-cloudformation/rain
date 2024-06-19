// Package codeartifact provides functions that interact with the AWS CodeArtifact API
package codeartifact

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codeartifact/types"
	"io"
	"io/fs"
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

// SemverIsGreater checks if a version string is greater than another version string
//
//	The version strings are in the format "major.minor.patch"
//	The function returns true if the first version string is greater than the second version string,
//	and false otherwise.
func SemverIsGreater(a, b string) (bool, error) {
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

func IncrementSemverMinorVersion(version string) (string, error) {
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

// GetLatestPackageVersion gets the latest version of a package
func GetLatestPackageVersion(packageInfo *PackageInfo) (string, error) {
	var latestVersion string
	client := getClient()

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
			return "", nil
		}
	} else if err != nil {
		return "", err
	}

	// Get the current version of the package
	config.Debugf("Package versions: %+v", res.Versions)

	// Iterate through the versions and find the latest version
	for _, version := range res.Versions {
		if version.Version == nil {
			continue // Shouldn't happen
		}
		if latestVersion == "" {
			latestVersion = *version.Version
			continue
		}
		isGreater, err := SemverIsGreater(*version.Version, latestVersion)
		if err != nil {
			return "", fmt.Errorf("unable to compare versions %s and %s: %v", *version.Version, latestVersion, err)
		}
		if isGreater {
			latestVersion = *version.Version
		}
	}

	config.Debugf("Latest version: %s", latestVersion)

	return latestVersion, nil
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

	// Convert the hash value to a hex string
	hashString := fmt.Sprintf("%x", hashValue)

	config.Debugf("Hash value: %x", hashString)

	var newVersion string

	// Reset the zipFile reader
	zipReader = bytes.NewReader(zipFile)

	if packageInfo.Version != "" {
		newVersion = packageInfo.Version

		// TODO: Confirm replacing the version if it already exists or is lower than the latest

	} else {
		latestVersion, err := GetLatestPackageVersion(packageInfo)
		if err != nil {
			return fmt.Errorf("failed to get latest package version: %v", err)
		}
		if latestVersion == "" {
			newVersion = DefaultVersion
		} else {
			// Increment the minor version number by 1
			// For example, version 1.2.3 becomes 1.3.0
			newVersion, err = IncrementSemverMinorVersion(latestVersion)
			if err != nil {
				return fmt.Errorf("unable to increment version %s: %v", latestVersion, err)
			}
		}
	}

	config.Debugf("About to publish new version: %s", newVersion)

	packageInfo.Version = newVersion

	// Call the codeartifact api to publish the package version
	res, err := client.PublishPackageVersion(context.Background(),
		&codeartifact.PublishPackageVersionInput{
			Domain:         aws.String(packageInfo.Domain),
			Repository:     aws.String(packageInfo.Repo),
			Package:        aws.String(packageInfo.Name),
			Format:         types.PackageFormatGeneric,
			Namespace:      aws.String(DefaultNamespace),
			PackageVersion: aws.String(newVersion),
			AssetContent:   zipReader,
			AssetSHA256:    aws.String(hashString),
			AssetName:      aws.String(packageInfo.Name + ".zip"),
		})

	if err != nil {
		return err
	}

	config.Debugf("Package version published: %+v", res)

	return nil
}

// Unzip unzips a zip file to a destination directory
func Unzip(f *os.File, dest string) error {
	// Open a file reader
	r, err := zip.OpenReader(f.Name())
	if err != nil {
		return err
	}
	defer func(r *zip.ReadCloser) {
		err := r.Close()
		if err != nil {
			config.Debugf("Error closing zip reader: %s", err)
		}
	}(r)

	// Iterate through the files in the archive,
	// extracting each to the output directory
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func(rc io.ReadCloser) {
			err := rc.Close()
			if err != nil {
				config.Debugf("Error closing file: %s", err)
			}
		}(rc)

		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			mode := fs.ModePerm
			err := os.MkdirAll(fpath, mode)
			if err != nil {
				return err
			}
			config.Debugf("Created directory: %s with mode %x", fpath, mode)
		} else {
			var fdir string
			if lastIndex := strings.LastIndex(fpath, string(os.PathSeparator)); lastIndex > -1 {
				fdir = fpath[:lastIndex]
				mode := fs.ModePerm
				err := os.MkdirAll(fdir, mode)
				if err != nil {
					return err
				}
				config.Debugf("Created subdirectory: %s with mode %x", fdir, mode)
			}

			f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func(f *os.File) {
				err := f.Close()
				if err != nil {
					config.Debugf("Error closing file: %s", err)
				}
			}(f)

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// GetAssetHashForPackage returns the hash of the asset for a package
func GetAssetHashForPackage(packageInfo *PackageInfo) (string, error) {

	client := getClient()

	// Call the api to describe the package version, so we can verify the hash
	assets, err := client.ListPackageVersionAssets(context.Background(),
		&codeartifact.ListPackageVersionAssetsInput{
			Domain:         aws.String(packageInfo.Domain),
			Repository:     aws.String(packageInfo.Repo),
			Package:        aws.String(packageInfo.Name),
			Format:         types.PackageFormatGeneric,
			Namespace:      aws.String(DefaultNamespace),
			PackageVersion: aws.String(packageInfo.Version),
		})

	if err != nil {
		return "", err
	}

	config.Debugf("Package version description: %+v", assets)

	if len(assets.Assets) == 0 {
		return "", fmt.Errorf("no assets found")
	}

	singleAsset := assets.Assets[0]
	if *singleAsset.Name != packageInfo.Name+".zip" {
		return "", fmt.Errorf("unexpected first asset name does not match: %s", *singleAsset.Name)
	}

	// Verify the hash of the asset content
	existingHash, ok := singleAsset.Hashes["SHA-256"]
	if !ok {
		return "", fmt.Errorf("no SHA-256 hash found in asset")
	}

	return existingHash, nil
}

func GetAsset(packageInfo *PackageInfo) (io.Reader, error) {
	client := getClient()

	// Call the api to get the asset content
	res, err := client.GetPackageVersionAsset(context.Background(),
		&codeartifact.GetPackageVersionAssetInput{
			Domain:         aws.String(packageInfo.Domain),
			Repository:     aws.String(packageInfo.Repo),
			Package:        aws.String(packageInfo.Name),
			Format:         types.PackageFormatGeneric,
			Namespace:      aws.String(DefaultNamespace),
			PackageVersion: aws.String(packageInfo.Version),
			Asset:          aws.String(packageInfo.Name + ".zip"),
		})

	if err != nil {
		spinner.Pop()
		return nil, err
	}

	config.Debugf("Package version: %+v", res)

	return res.Asset, nil
}
