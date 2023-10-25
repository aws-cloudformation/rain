package ccdeploy

import (
	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/s11n"
)

// update compares the template with the current state and returns a
// template annotated with operations to perform on each resource
func update(stateTemplate cft.Template, template cft.Template) (cft.Template, error) {

	// Create a diff between the current state and template
	d := diff.New(stateTemplate, template)
	config.Debugf("update diff:\nMode:%v\n%v", d.Mode(), d.Format(true))

	// Each modified resource needs to be tagged with create-update-delete-none,
	// so that deployResource knows which action to take.
	// We don't need a deep diff, only to identify what has changed.

	// In the template, write a node to the resource's State
	/*
		   Resources:
			 MyBucket:
				Type: AWS::S3::Bucket

			State:
			  ResourceModels:
			    MyBucket:
				  Action: Create or Update or Delete or None

	*/

	// Iterate through the resources and check the diff
	rootMap := stateTemplate.Node.Content[0]
	_, resourceMap := s11n.GetMapValue(rootMap, "Resources")
	if resourceMap == nil {
		panic("Expected to find a Resources section in the state template")
	}
	for i, r := range resourceMap.Content {
		if i%2 == 0 {
			name := r.Value
			action := diff.ResourceAction(d, name)
			config.Debugf("update action for %v is %v", name, action)
			// TODO
		}
	}
	// TODO - Look in the other template for new resources
	// Or iterate through the diff instead since it has everything
	// maybe diff.GetResourceActions -> map[string]ActionType

	return stateTemplate, nil
}
