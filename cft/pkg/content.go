package pkg

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/config"
)

type ModuleContent struct {
	Content    []byte
	NewRootDir string
	BaseUri    string
}



func isHttpsUrl(uri string) bool {
	return strings.HasPrefix(uri, "https://")
}

func isS3URI(uri string) bool {
	return strings.HasPrefix(uri, "s3://")
}

// Get the module's content from a local file, memory, or a remote uri
func getModuleContent(
	root string,
	t *cft.Template,
	templateFiles *embed.FS,
	baseUri string,
	uri string) (*ModuleContent, error) {

	config.Debugf("getModuleContent root: %s, uri: %s", root, uri)

	var content []byte
	var err error
	var newRootDir string

	// Check to see if this is an alias like "$alias/foo.yaml" (new format)
	isZip := false
	if strings.HasPrefix(uri, "$") {
		parts := strings.SplitN(uri, "/", 2)
		if len(parts) == 2 {
			alias := parts[0][1:] // Remove the $ prefix
			path := parts[1]

			if t.Packages != nil {
				if packageAlias, ok := t.Packages[alias]; ok {

					if strings.HasSuffix(packageAlias.Location, ".zip") {
						isZip = true
						
						// Use DownloadFromZip directly
						zipLocation := packageAlias.Location
						// For local files, resolve the path relative to the template's directory
						if !isS3URI(zipLocation) && !isHttpsUrl(zipLocation) && !filepath.IsAbs(zipLocation) {
							zipLocation = filepath.Join(root, zipLocation)
						}
						
						content, err = DownloadFromZip(zipLocation, packageAlias.Hash, path)
						if err != nil {
							return nil, err
						}
					} else {
						// Replace the alias with the actual location
						uri = packageAlias.Location + "/" + path
					}
				} else {
					config.Debugf("Package alias not found: %s", alias)
				}
			} else {
				config.Debugf("No packages defined in template")
			}
		}
	}

	// Look for a zip path where we already fixed the $alias
	// getModuleContent: root=cft/pkg/tmpl/awscli-modules, baseUri=, uri=package.zip/zip-module.yaml
	if strings.Contains(uri, ".zip/") {
		isZip = true
		
		// Extract the zip location and path within the zip
		zipIndex := strings.Index(uri, ".zip/")
		if zipIndex > 0 {
			zipLocation := uri[:zipIndex+4]  // Include the .zip part
			zipPath := uri[zipIndex+5:]      // Skip the .zip/ part
			
			// For local files, resolve the path relative to the template's directory
			if !isS3URI(zipLocation) && !isHttpsUrl(zipLocation) && !filepath.IsAbs(zipLocation) {
				zipLocation = filepath.Join(root, zipLocation)
			}
			
			config.Debugf("Extracting from zip: %s, path: %s", zipLocation, zipPath)
			
			// Use DownloadFromZip directly - it can handle S3, HTTPS, and local files
			content, err = DownloadFromZip(zipLocation, "", zipPath)
			if err != nil {
				return nil, err
			}
		}
	}

	// Check to see if this is an alias like "alias/foo.yaml" (legacy format)
	if !isZip {
		packageAlias := checkPackageAlias(t, uri)
		if packageAlias != nil {

			path := strings.Replace(uri, packageAlias.Alias+"/", "", 1)
			if strings.HasSuffix(packageAlias.Location, ".zip") {
				isZip = true
				
				// Use DownloadFromZip directly
				zipLocation := packageAlias.Location
				// For local files, resolve the path relative to the template's directory
				if !isS3URI(zipLocation) && !isHttpsUrl(zipLocation) && !filepath.IsAbs(zipLocation) {
					zipLocation = filepath.Join(root, zipLocation)
				}
				
				content, err = DownloadFromZip(zipLocation, packageAlias.Hash, path)
				if err != nil {
					return nil, err
				}
			} else {
				uri = strings.Replace(uri, packageAlias.Alias, packageAlias.Location, 1)
			}
		}
	}

	// Is this a local file or a URL or did we already unzip a package?
	if isZip {
		config.Debugf("Using content from a zipped module package (length: %d bytes)", len(content))
	} else if isHttpsUrl(uri) || isS3URI(uri) {
		config.Debugf("Downloading from URL: %s", uri)
		content, err = downloadModule(uri)
		if err != nil {
			return nil, err
		}

		// Once we see a URL instead of a relative local path,
		// we need to remember the base URL so that we can
		// fix relative paths in any referenced modules.

		// Strip the file name from the uri
		urlParts := strings.Split(uri, "/")
		baseUri = strings.Join(urlParts[:len(urlParts)-1], "/")

	} else {
		config.Debugf("Downloading from a local file, baseUri=%s, uri=%s", baseUri, uri)
		if baseUri != "" {
			// If we have a base URL, prepend it to the relative path
			uri = baseUri + "/" + uri
			content, err = downloadModule(uri)
			if err != nil {
				return nil, err
			}
		} else if templateFiles != nil {
			// Read from the embedded file system (for the build -r command)
			// We have to hack this since embed doesn't understand "path/../"
			embeddedPath := strings.Replace(root, "../", "", 1) +
				"/" + strings.Replace(uri, "../", "", 1)

			content, err = templateFiles.ReadFile(embeddedPath)
			if err != nil {
				return nil, err
			}
			newRootDir = filepath.Dir(embeddedPath)
		} else {
			// Read the local file
			path := uri
			if !filepath.IsAbs(path) {
				path = filepath.Join(root, path)
			}

			info, err := os.Stat(path)
			if err != nil {
				config.Debugf("Error accessing file: %v", err)
				return nil, err
			}

			if info.IsDir() {
				return nil, fmt.Errorf("'%s' is a directory", path)
			}

			content, err = os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			newRootDir = filepath.Dir(path)
		}
	}

	return &ModuleContent{content, newRootDir, baseUri}, nil
}
