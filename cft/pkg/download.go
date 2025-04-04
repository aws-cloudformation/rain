package pkg

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/google/uuid"
)

// Downloads the hash file and returns the contents
func downloadHash(uri string) (string, error) {

	config.Debugf("Downloading %s", uri)
	resp, err := http.Get(uri)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			config.Debugf("Error closing body: %v", err)
		}
	}(resp.Body)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil
	}

	retval := string(data)
	retval = strings.Trim(retval, " \n")
	return retval, nil
}

// DownloadFromZip retrieves a single file from a zip file hosted on a URI
func DownloadFromZip(uriString string, verifyHash string, path string) ([]byte, error) {
	var zipData []byte
	var err error
	
	// Check if it's a URL or local file
	if strings.HasPrefix(uriString, "http://") || strings.HasPrefix(uriString, "https://") {
		// Download from URL
		config.Debugf("Downloading %s", uriString)
		resp, err := http.Get(uriString)
		if err != nil {
			return nil, err
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				config.Debugf("Error closing body: %v", err)
			}
		}(resp.Body)
		
		zipData, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	} else {
		// Read local file
		config.Debugf("Reading local file %s", uriString)
		zipData, err = os.ReadFile(uriString)
		if err != nil {
			return nil, err
		}
	}

	// Save the zip data to a temp file
	pFile, err := os.CreateTemp("", "rain-package-*.zip")
	if err != nil {
		return nil, err
	}
	defer func(pFile *os.File) {
		err := pFile.Close()
		if err != nil {
			config.Debugf("Error closing file: %s", err)
		}
		// Clean up the temp file
		os.Remove(pFile.Name())
	}(pFile)

	config.Debugf("Saving zip content to %s", pFile.Name())

	// Write the zip data to the temp file
	if _, err := pFile.Write(zipData); err != nil {
		return nil, err
	}

	// Seek to the beginning of the file
	if _, err := pFile.Seek(0, 0); err != nil {
		return nil, err
	}

	if verifyHash != "" {
		// Create a sha256 hash of the asset content and verify it
		hash := sha256.New()
		if _, err := hash.Write(zipData); err != nil {
			return nil, err
		}
		hashValue := hash.Sum(nil)

		// Convert the hash value to a hex string
		hashString := fmt.Sprintf("%x", hashValue)

		// Download or read the hash
		var originalHash string
		if strings.HasPrefix(verifyHash, "http://") || strings.HasPrefix(verifyHash, "https://") {
			originalHash, err = downloadHash(verifyHash)
			if err != nil {
				return nil, err
			}
		} else {
			hashData, err := os.ReadFile(verifyHash)
			if err != nil {
				return nil, err
			}
			originalHash = strings.TrimSpace(string(hashData))
		}

		if originalHash != hashString {
			return nil, fmt.Errorf("hash does not match: %s != %s", originalHash, hashString)
		}
	}

	// Unzip the temp file
	dir := filepath.Join(os.TempDir(), uuid.NewString())
	err = Unzip(pFile, dir)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir) // Clean up the temp directory

	// Read the requested file from the unzipped directory
	filePath := filepath.Join(dir, path)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s from zip: %v", path, err)
	}

	return content, nil
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
		} else {
			var fdir string
			if lastIndex := strings.LastIndex(fpath, string(os.PathSeparator)); lastIndex > -1 {
				fdir = fpath[:lastIndex]
				mode := fs.ModePerm
				err := os.MkdirAll(fdir, mode)
				if err != nil {
					return err
				}
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

// downloadModule downloads the file from the given URI and returns its content as a byte slice.
func downloadModule(uri string) ([]byte, error) {
	config.Debugf("Downloading %s", uri)
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			config.Debugf("Error closing body: %v", err)
		}
	}(resp.Body)

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return content, nil
}
