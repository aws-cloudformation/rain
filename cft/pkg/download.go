package pkg

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
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

	u, err := url.Parse(uriString)
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(u.Path)

	// Save the asset content to a temp file
	pFile, err := os.CreateTemp("", filename)
	if err != nil {
		return nil, err
	}
	defer func(pFile *os.File) {
		err := pFile.Close()
		if err != nil {
			config.Debugf("Error closing file: %s", err)
		}
	}(pFile)

	config.Debugf("Saving zip content to %s", pFile.Name())

	// Write the asset content to the temp file
	if _, err := io.Copy(pFile, resp.Body); err != nil {
		return nil, err
	}

	// Seek to the beginning of the file
	if _, err := pFile.Seek(0, 0); err != nil {
		return nil, err
	}

	if verifyHash != "" {
		// Create a sha256 hash of the asset content and verify it
		hash := sha256.New()
		// Read the contents of the temporary pFile and generate a sha256 hash
		if _, err := io.Copy(hash, pFile); err != nil {
			return nil, err
		}
		hashValue := hash.Sum(nil)

		// Convert the hash value to a hex string
		hashString := fmt.Sprintf("%x", hashValue)

		// Reset pFile to the beginning
		if _, err := pFile.Seek(0, 0); err != nil {
			return nil, err
		}

		// Download the hash
		originalHash, err := downloadHash(verifyHash)
		if err != nil {
			return nil, err
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

	content, err := os.ReadFile(filepath.Join(dir, path))
	if err != nil {
		return nil, err
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
