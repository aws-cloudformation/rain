package ccdeploy

import (
	"fmt"
	"time"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/graph"
	"github.com/aws-cloudformation/rain/internal/aws/ccapi"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// getTemplateResource returns the yaml node based on the logical id
func getTemplateResource(logicalId string) (*yaml.Node, error) {
	rootMap := template.Node.Content[0]
	_, resources := s11n.GetMapValue(rootMap, "Resources")
	if resources == nil {
		panic("Expected to find a Resources section in the template")
	}
	for i, r := range resources.Content {
		if i%2 != 0 {
			continue
		}
		if logicalId == r.Value {
			resource := resources.Content[i+1]
			return resource, nil
		}
	}
	return nil, fmt.Errorf("could not find Resource %v", logicalId)
}

// deployResource calls the Cloud Control API to deploy the resource
func deployResource(resource *Resource) {
	config.Debugf("Deploying %v...", resource)

	resource.State = Deploying

	// TODO - Resolve instrinsics before creating the resource
	// This will depend on the post-deployment state of dependencies
	resolvedNode, err := resolve(resource)
	if err != nil {
		panic(err)
	}

	switch resource.Action {
	case Create:

		// Get the properties and call ccapi
		var identifier string
		var model string
		identifier, model, err = ccapi.CreateResource(resource.Name, resolvedNode)
		if err != nil {
			config.Debugf("deployResource failed: %v", err)
			resource.State = Failed
			resource.Message = fmt.Sprintf("%v", err)
		} else {
			resource.State = Deployed
			resource.Message = "Success"
			resource.Identifier = identifier
			resource.Model = model
		}
	case Update:

		config.Debugf("deployResource Update TODO")

	case Delete:

		config.Debugf("deployResource Delete TODO")
	default:
		// None means this is an update with no change to the model
		config.Debugf("deployResource not deploying unchanged %v", resource.Name)
	}

}

// ready returns true if the resource has no undeployed dependencies
func ready(resource *Resource, g *graph.Graph) bool {

	// Iterate over each of this resource's dependencies
	for _, dep := range g.Get(graph.Node{Name: resource.Name, Type: "Resources"}) {

		if dep.Type != "Resources" {
			continue
		}

		depr := resMap[dep.Name]

		// If the dependency is not deployed, terminate
		if depr.State != Deployed {
			return false
		}

		// Recurse on each dependency
		if !ready(depr, g) {
			return false
		}
	}

	// If we get here, the resource can be deployed
	return true
}

// DeploymentResults captures everything that happened as a result of deployment
type DeploymentResults struct {
	Succeeded bool
	State     cft.Template
	Resources map[string]*Resource
}

// deployTemplate deloys the CloudFormation template using the Cloud Control API
// A failed deployment will result in DeploymentResults.Succeeded = false
// A non-nil error is returned when something unexpected caused a failure
// not related to actually deploying resources, like an invalid template
func deployTemplate(template cft.Template) (*DeploymentResults, error) {

	results := &DeploymentResults{
		Succeeded: true,
		State:     cft.Template{},
		Resources: make(map[string]*Resource),
	}

	results.State.Node = node.Clone(template.Node)

	// Create a dependency graph of the template
	g := graph.New(template)
	nodes := g.Nodes()

	/*
		Downwards is "depends on"

		   A   E   F
		  / \   \\
		 B   C   GH
			  \
			   D

		B, D, G, and H can all be deployed at the same time.

		We work our way up from the bottom, deploying resources concurrently
		as soon as they have no more undeployed dependencies.

		Deletes have to go in the reverse order.
		A depends on B, if I'm deleting both, A has to be deleted first.

		TODO: Make a separate graph for deletes, do them all first.
		Verify that we are not deleting anything depended on by a live resource.
	*/

	// Wrap Nodes in a Resource to add state
	resources := make([]*Resource, 0)
	for _, n := range nodes {
		if n.Type == "Resources" {
			y, err := getTemplateResource(n.Name)
			if err != nil {
				panic(fmt.Sprintf("%v not found", n.Name))
			}

			_, typeNode := s11n.GetMapValue(y, "Type")
			if typeNode == nil {
				return nil, fmt.Errorf("expected resource %v to have a Type", n.Name)
			}
			typeName := typeNode.Value

			// Determine if this is a create, update, or delete
			var action ActionType
			_, stateNode := s11n.GetMapValue(y, "State")
			if stateNode == nil {
				// Assume this is a new deployment
				action = Create
			} else {
				for i, s := range stateNode.Content {
					if s.Value == "Action" {
						a := stateNode.Content[i+1].Value
						action = ActionType(a)
						isValid := false
						switch action {
						case Create, Update, Delete, None:
							isValid = true
						}
						if !isValid {
							return nil, fmt.Errorf("invalid Action %v for %v", a, n.Name)
						}
					}
				}
			}

			r := NewResource(n.Name, typeName, Waiting, y)
			r.Action = action
			resources = append(resources, r)
		}
	}

	numResources := len(resources)
	numDone := 0
	failed := false

	config.Debugf("About to deploy %v resources", numResources)

	for numDone < numResources {

		config.Debugf("Starting an iteration over resources (%v/%v done)",
			numDone, numResources)

		numDone = 0

		for _, r := range resources {

			if r.State == Deployed || r.State == Canceled {
				numDone += 1
				continue
			}

			if r.State == Deploying {
				continue
			}

			if r.State == Failed {
				// This will prevent any additional resources from deploying,
				// but any that had already started creation will complete
				failed = true
				numDone += 1
				continue
			}

			if !failed {
				// Recurse dependencies to see if it's ok to deploy this one now
				if ready(r, &g) {
					// Start a goroutine to do the actual deployment
					go deployResource(r)
				}
			} else {
				if r.State == Waiting {
					r.State = Canceled
				}
			}
		}

		for _, r := range resources {
			config.Debugf("%v", r)
		}

		// Give deployment routines time to finish
		// TODO: We could be smarter about this with channels...
		time.Sleep(time.Second * 1)
	}

	for _, r := range resources {
		results.Resources[r.Name] = r
	}

	if failed {
		results.Succeeded = false
	}

	return results, nil

}
