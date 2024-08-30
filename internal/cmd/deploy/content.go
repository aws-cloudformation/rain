package deploy

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
)

// processMetadata looks for Rain command in resource Metadata
// For CREATE and UPDATE operations, a Content node on a bucket
// will upload local assets to the bucket.
func processMetadata(template cft.Template, stackName string) error {

	// For some reason Package created an extra document node
	// (And CreateChangeSet is ok with this...?)
	template.Node = template.Node.Content[0]

	// Iterate over resources looking for buckets
	resources, err := template.GetSection(cft.Resources)
	if err != nil {
		return err
	}

	for i := 0; i < len(resources.Content); i += 2 {
		logicalId := resources.Content[i].Value
		bucket := resources.Content[i+1]
		_, typ, _ := s11n.GetMapValue(bucket, "Type")
		if typ == nil {
			continue
		}
		if typ.Value != "AWS::S3::Bucket" {
			continue
		}

		config.Debugf("processMetadata bucket: %s \n%v", logicalId, node.ToSJson(bucket))
		_, n, _ := s11n.GetMapValue(bucket, "Metadata")
		if n == nil {
			continue
		}
		config.Debugf("processMetadata found Metadata")
		_, n, _ = s11n.GetMapValue(n, "Rain")
		if n == nil {
			continue
		}
		_, contentPath, _ := s11n.GetMapValue(n, "Content")
		if contentPath == nil {
			continue
		}
		config.Debugf("processMetadata found contentPath for resource: %s",
			contentPath.Value)

		// Get the bucket name
		sr, err := cfn.GetStackResource(stackName, logicalId)
		if err != nil {
			return err
		}
		bucketName := *sr.PhysicalResourceId
		config.Debugf("processMetadata bucket %s name is %s", logicalId, bucketName)

		// Assume contentPath.Value is a directory and Put all files
		// TODO: Add options for a prefix, zip, single file

		p := contentPath.Value

		// TODO: Console output for progress

		// Recursively walk the directory and upload all files
		err = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				config.Debugf("error walking %s: %v", p, err)
				return err
			}
			if !info.IsDir() {
				f, readErr := os.ReadFile(path)
				if readErr != nil {
					config.Debugf("error reading %s: %v", path, err)
					return readErr
				}
				// Get rid of the local directory path
				// For example, if the local file is myfiles/foo/bar.txt,
				// put bar.txt into the bucket
				putPath := strings.Replace(path, p, "", 1)
				putPath = strings.TrimLeft(putPath, "/")
				putErr := s3.PutObject(bucketName, putPath, f)
				config.Debugf("PutObject: %s/%s", bucketName, putPath)
				if putErr != nil {
					config.Debugf("error putting %s/%s: %v", bucketName, putPath, putErr)
					return putErr
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil

}
