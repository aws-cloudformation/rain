package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/cloudfront"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/s11n"
)

// processMetadataAfter looks for Rain commands in resource Metadata
// that need to be run after deployment.
// For CREATE and UPDATE operations, a Content node on a bucket
// will upload local assets to the bucket.
func processMetadataAfter(template cft.Template, stackName string, rootDir string) error {

	// For some reason Package created an extra document node
	// (And CreateChangeSet is ok with this...?)
	template.Node = template.Node.Content[0]

	buckets := template.GetResourcesOfType("AWS::S3::Bucket")
	for _, bucket := range buckets {
		logicalId := bucket.LogicalId
		_, n, _ := s11n.GetMapValue(bucket.Node, "Metadata")
		if n == nil {
			continue
		}
		_, n, _ = s11n.GetMapValue(n, "Rain")
		if n == nil {
			continue
		}

		_, contentPath, _ := s11n.GetMapValue(n, "Content")
		if contentPath == nil {
			continue
		}

		// Ignore RAIN_NO_CONTENT or an empty string
		if contentPath.Value == "" || contentPath.Value == "RAIN_NO_CONTENT" {
			continue
		}

		_, run, _ := s11n.GetMapValue(n, "Run")
		if run != nil && run.Value != "" {
			// Run a script before uploading the content
			config.Debugf("Running %s before uploading content", run.Value)
			relativePath := filepath.Join(".", rootDir, run.Value)
			absPath, absErr := filepath.Abs(relativePath)
			if absErr != nil {
				config.Debugf("filepath.Abs failed? %s", absErr)
				return absErr
			}
			cmd := exec.Command(absPath)
			var stdout strings.Builder
			var stderr strings.Builder
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			cmd.Dir = rootDir
			err := cmd.Run()
			if err != nil {
				config.Debugf("Content Run %s failed with %s: %s",
					run.Value, err, stderr.String())
				// TODO: Better error message when not debugging!
				return err
			}
		} else {
			config.Debugf("Run not found")
		}

		// Assume contentPath.Value is a directory and Put all files
		p := filepath.Join(rootDir, contentPath.Value)

		// Get the bucket name
		sr, err := cfn.GetStackResource(stackName, logicalId)
		if err != nil {
			return err
		}
		bucketName := *sr.PhysicalResourceId

		spinner.Push(fmt.Sprintf("Uploading the contents of %s to %s", p, bucketName))

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
		spinner.Pop()
		if err != nil {
			return err
		}

		// Look for a LogicalId of a Distribution to invalidate
		_, dlid, _ := s11n.GetMapValue(n, "DistributionLogicalId")
		if dlid != nil {

			spinner.Push(fmt.Sprintf("Invalidating CloudFront Distribution %s", dlid.Value))

			// Look up the distribution id
			sr, err := cfn.GetStackResource(stackName, dlid.Value)
			if err != nil {
				return err
			}
			did := *sr.PhysicalResourceId
			config.Debugf("About to invalidate %s", did)

			// Invalidate the distribution
			err = cloudfront.Invalidate(did)
			spinner.Pop()
			if err != nil {
				return err
			}
		}
	}

	return nil

}
