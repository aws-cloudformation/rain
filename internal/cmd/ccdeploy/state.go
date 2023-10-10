package ccdeploy

import "github.com/aws-cloudformation/rain/cft"

// checkState looks for an existing state file.
// If one does not exist, it is created.
// If one exists and there is a lock file, an error is returned
//
//	unless this process owns the lock file...
//
// If one exists and there is no lock file, this is an update
func checkState(name string, template cft.Template, bucketName string) (cft.Template, error) {
	return template, nil // TODO
}
