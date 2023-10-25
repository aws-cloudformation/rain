package ccdeploy

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// update compares the template with the current state and returns a
// cloned template annotated with operations to perform on each resource
func update(stateTemplate cft.Template, template cft.Template) (cft.Template, error) {

	// Create a diff between the current state and template
	d := diff.New(stateTemplate, template)
	config.Debugf("update diff:\nMode:%v\n%v", d.Mode(), d.Format(true))

	// Each modified resource needs to be tagged with create-update-delete-none,
	// so that deployResource knows which action to take.
	// We don't need a deep diff, only to identify what resources have changed.

	// In the template, write a node to the resource's State
	/*
		   Resources:
			 MyBucket:
				Type: AWS::S3::Bucket
			 State:
				Action: Create or Update or Delete or None
				Identifier: ...
				ResourceModel: ? Do we need this for ccapi update? Drift detection?

	*/

	// Figure out what we're doing with each resource (create, update, delete, nothing)
	actions := diff.GetResourceActions(d)

	// Iterate through the state resources and check the diff
	stateRootMap := stateTemplate.Node.Content[0]
	_, stateResourceMap := s11n.GetMapValue(stateRootMap, "Resources")
	if stateResourceMap == nil {
		panic("Expected to find a Resources section in the state template")
	}

	// Make a copy of the template so the caller still has the original as the user wrote it
	newTemplate := cft.Template{}
	newTemplate.Node = node.Clone(template.Node)

	// Get a reference to the resources in the new template
	newRootMap := newTemplate.Node.Content[0]
	_, newResourceMap := s11n.GetMapValue(newRootMap, "Resources")
	if newResourceMap == nil {
		panic("Expected to find a Resources section in the new template")
	}

	stateResources := make(map[string]*yaml.Node, 0)
	newResources := make(map[string]*yaml.Node, 0)
	resourceActionStates := make(map[string]*yaml.Node) // "State" mapping node

	for i, r := range stateResourceMap.Content {
		if i%2 == 0 {
			name := r.Value
			stateResources[name] = stateResourceMap.Content[i+1]
		}
	}
	for i, r := range newResourceMap.Content {
		if i%2 == 0 {
			name := r.Value
			newResources[name] = newResourceMap.Content[i+1]
			resourceActionStates[name] = addMap(newResources[name], "State")
		}
	}

	for k, v := range actions {
		rmap, ok := resourceActionStates[k]
		if !ok {
			// Anything missing from the new template is a removed resource to be deleted
			// Clone the resource from the state template and re-add it to the
			// new template with Action: Delete
			if v != diff.Delete {
				return stateTemplate, fmt.Errorf("unexpected missing resource %v has Action %v", k, v)
			}
			newResourceMap.Content = append(newResourceMap.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: k})
			cloned := node.Clone(stateResources[k])
			clonedStateMap := addMap(cloned, "State")
			add(clonedStateMap, "Action", string(v))
			// TODO: Add the identifier from the state file so we know what to delete
			newResourceMap.Content = append(newResourceMap.Content, cloned)
		} else {
			// Create, Update, None
			add(rmap, "Action", string(v))
		}
	}

	// TODO - What about drift? We should check the resource model for differences.
	// How do we handle that? Ask to apply it to the template?
	// Ask to undo the drift?

	config.Debugf("About to return new template:\n%v",
		format.String(newTemplate, format.Options{JSON: false, Unsorted: false}))

	return newTemplate, nil
}
