package ccdeploy

import "github.com/aws-cloudformation/rain/cft"

// checkState looks for an existing state file.
// If one does not exist, it is created.
// If one exists and there is  a lock file, an error is returned
// If one exists and there is no lock file, this is an update
func checkState(name string, template cft.Template) (cft.Template, error) {
	return nil, nil // TODO
}
