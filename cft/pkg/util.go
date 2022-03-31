// Package pkg provides functionality similar to the AWS CLI cloudformation package command
// but has greater flexibility, allowing content to be included anywhere in a template
//
// To include content into your templates, use any of the following either as YAML tags
// or as one-property objects, much as AWS instrinsic functions are used, e.g. "Fn::Join"
//
// `Rain::Include`: insert the content of the file into the template directly. The file must be in YAML or JSON format.
// `Rain::Embed`: insert the content of the file as a string
// `Rain::S3Http`: uploads the file or directory (zipping it first) to S3 and returns the HTTP URI (i.e. `https://bucket.s3.region.amazonaws.com/key`)
// `Rain::S3`: a string value uploads the file or directory (zipping it first) to S3 and returns the S3 URI (i.e. `s3://bucket/key`)
// `Rain::S3`: an object with the following properties
//    `Path`: path to the file or directory. If a directory is supplied, it will be zipped before uploading to S3
//    `BucketProperty`: Name of returned property that will contain the bucket name
//    `KeyProperty`: Name of returned property that will contain the object key
//    `VersionProperty`: (optional) Name of returned property that will contain the object version
package pkg

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/aws-cloudformation/rain/internal/aws"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"gopkg.in/yaml.v3"
)

type s3Path struct {
	bucket string
	key    string
	region string
}

func (s *s3Path) URI() string {
	return fmt.Sprintf("s3://%s/%s", s.bucket, s.key)
}

func (s *s3Path) HTTP() string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, s.key)
}

var uploads = map[string]*s3Path{}

func zipPath(root string) (string, error) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "*.zip")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	w := zip.NewWriter(tmpFile)
	defer w.Close()

	zRoot := root
	info, err := os.Stat(zRoot)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		zRoot = filepath.Dir(zRoot)
	}

	err = filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		zPath, err := filepath.Rel(zRoot, path)
		if err != nil {
			return err
		}

		zPath = filepath.ToSlash(zPath)

		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		fh.Name = zPath
		fh.Method = zip.Deflate

		out, err := w.CreateHeader(fh)
		if err != nil {
			return err
		}

		_, err = io.Copy(out, in)
		return err
	})

	return tmpFile.Name(), err
}

// Upload a file or directory to S3.
// If path is a directory, it will be zipped first.
func upload(root, path string, force bool) (*s3Path, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}

	artifactName := path
	if force {
		artifactName = "zip:" + artifactName
	}

	if result, ok := uploads[artifactName]; ok {
		config.Debugf("Using existing upload for: %s\n", path)
		return result, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.IsDir() || force {
		// Zip it!
		zipped, err := zipPath(path)
		if err != nil {
			return nil, err
		}
		config.Debugf("Zipped %s as %s\n", path, zipped)
		path = zipped
	}

	config.Debugf("Uploading: %s\n", path)

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	bucket := s3.RainBucket(false)
	key, err := s3.Upload(bucket, content)

	uploads[artifactName] = &s3Path{
		bucket: bucket,
		key:    key,
		region: aws.Config().Region,
	}

	return uploads[artifactName], err
}

func expectString(n *yaml.Node) (string, error) {
	if len(n.Content) != 2 {
		return "", fmt.Errorf("expected a mapping node")
	}

	if n.Content[1].Kind != yaml.ScalarNode {
		return "", fmt.Errorf("expected a scalar value")
	}

	return n.Content[1].Value, nil
}

func expectFile(n *yaml.Node, root string) ([]byte, string, error) {
	path, err := expectString(n)
	if err != nil {
		return nil, "", err
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, path, err
	}

	if info.IsDir() {
		return nil, path, fmt.Errorf("'%s' is a directory", path)
	}

	content, err := ioutil.ReadFile(path)

	return content, path, err
}

/*
func expectProps(n *yaml.Node, names ...string) (map[string]string, bool) {
	if len(n.Content) != 2 {
		return nil, false
	}

	if n.Content[1].Kind != yaml.MappingNode {
		return nil, false
	}

	var out map[string]interface{}

	err := n.Content[1].Decode(&out)
	if err != nil {
		return nil, false
	}

	props := make(map[string]string)

	for _, name := range names {
		value, exists := out[name]
		if !exists {
			return nil, false
		}

		str, ok := value.(string)
		if !ok {
			return nil, false
		}

		props[name] = str
	}

	return props, true
}
*/
