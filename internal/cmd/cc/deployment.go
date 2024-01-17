package cc

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/graph"
	"github.com/aws-cloudformation/rain/internal/aws/ccapi"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// getTemplateResource returns the yaml node based on the logical id
func getTemplateResource(template cft.Template, logicalId string) (*yaml.Node, error) {
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
	resource.Start = time.Now()
	defer func() { resource.End = time.Now() }()

	// Resolve instrinsics before creating the resource.
	// This depends on the post-deployment state of dependencies
	var resolvedNode *yaml.Node
	var err error
	if resource.Action == diff.Delete {
		// We don't need to resolve resources we are deleting.
		resolvedNode = resource.Node
	} else {
		resolvedNode, err = Resolve(resource)
		// TODO: Failing due to empty Model on updates
		// If the dependencies didn't change, they don't get deployed,
		// so we need to get the Model from the state file.
		// Or.. query ccapi for the live state?
		if err != nil {
			config.Debugf("deployResource resolve failed: %v", err)
			resource.State = Failed
			resource.Message = fmt.Sprintf("%v", err)
			return
		}
	}

	switch resource.Action {
	case diff.Create:

		// Get the properties and call ccapi
		var identifier string
		var model string
		identifier, model, err = ccapi.CreateResource(resource.Name, resolvedNode)
		if err != nil {
			config.Debugf("deployResource create failed: %v", err)
			resource.State = Failed
			resource.Message = fmt.Sprintf("%v", err)
		} else {
			resource.State = Deployed
			resource.Message = "Success"
			resource.Identifier = identifier
			resource.Model = model
		}
	case diff.Update:

		// // First get the current state of the resource
		// priorModel, err := ccapi.GetResource(resource.Identifier, resource.Type)
		// if err != nil {
		// 	config.Debugf("deployResource update get prior failed: %v", err)
		// 	resource.State = Failed
		// 	resource.Message = fmt.Sprintf("%v", err)
		// }
		//
		// We would need that at an earlier step for drift detection

		priorJson := resource.PriorJson

		var model string
		model, err = ccapi.UpdateResource(resource.Name,
			resource.Identifier, resolvedNode, priorJson)
		if err != nil {
			config.Debugf("deployResource update failed: %v", err)
			resource.State = Failed
			resource.Message = fmt.Sprintf("%v", err)
		} else {
			config.Debugf("deployResource update succeeded: %v", model)
			resource.State = Deployed
			resource.Message = "Success"
			resource.Model = model
		}

	case diff.Delete:

		err = ccapi.DeleteResource(resource.Name, resource.Identifier, resolvedNode)
		if err != nil {
			config.Debugf("deployResource delete failed: %v", err)
			resource.State = Failed
			resource.Message = fmt.Sprintf("%v", err)
		} else {
			resource.State = Deployed
			resource.Message = "Success"
		}

	default:
		// None means this is an update with no change to the model
		config.Debugf("deployResource not deploying unchanged %v. Identifier: %v, Model: %v",
			resource.Name, resource.Identifier, resource.Model)
		resource.State = Deployed
		resource.Message = "Success"

		// TODO: Are we missing the Model here?
	}

}

// ready returns true if the resource has no undeployed dependencies,
// TODO: unless the Action is Delete, in which case it returns true if it has
// no undeleted dependents
func ready(resource *Resource, g *graph.Graph) bool {

	node := graph.Node{Name: resource.Name, Type: "Resources"}
	var deps []graph.Node
	if resource.Action == diff.Delete {
		deps = g.GetReverse(node)
	} else {
		deps = g.Get(node)
	}

	// Iterate over each of this resource's dependencies (or dependents for deletes)
	for _, dep := range deps {

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

// Summarize prints out a summary of deployment results
func (results *DeploymentResults) Summarize() {
	for _, resource := range results.Resources {
		fmt.Println()
		fmt.Printf("%v\n", resource)
	}
}

// canDelete returns true if the resource can be deleted.
// This is not the same as the ready function, which tells
// you if it can be deleted "right now". This function looks
// for any dependent resources that are not marked for deletion
func canDelete(resource *Resource, g *graph.Graph, resourceMap map[string]*Resource) bool {

	dependents := g.GetReverse(graph.Node{Name: resource.Name, Type: "Resources"})
	for _, n := range dependents {
		depResource, ok := resourceMap[n.Name]
		if !ok {
			// This should not happen
			panic(fmt.Errorf("did not find %v in resourceMap", n.Name))
		}
		if depResource.Action != diff.Delete {
			return false
		}
		// Recurse
		if !canDelete(depResource, g, resourceMap) {
			return false
		}
	}
	return true
}

// verifyDeletes returns an error if any of the resources have dependents in the graph that are
// not also being deleted. All resources passed in must have Action = Delete
func verifyDeletes(resources []*Resource, g *graph.Graph, resourceMap map[string]*Resource) error {

	/*
				Down is depends on

				         A
				        / \
				       B   C
		                    \
		                     D

				If we are deleting C or D, but not A, fail.
	*/

	for _, resource := range resources {
		if resource.Action != diff.Delete {
			return fmt.Errorf("cannot verify deletes on resource %v that is not being deleted", resource.Name)
		}
		if !canDelete(resource, g, resourceMap) {
			return fmt.Errorf("resource %v has dependent resources that will not be deleted", resource.Name)
		}
	}

	return nil
}

// deployResources deploys a set of resources - either all the deletes, or
// all of the creates and updates. Deletes are handled in reverse dependency order.
func deployResources(resources []*Resource, results *DeploymentResults, g *graph.Graph) error {

	numResources := len(resources)
	numDone := 0
	failed := false

	config.Debugf("About to deploy %v resources", numResources)

	// TODO - Instead of re-evaluating readiness for each resource in a loop,
	// it might be better to create a deployment plan, with a pre-determined
	// order of operations, and then simply iterate over that plan.

	for numDone < numResources {

		// config.Debugf("Starting an iteration over resources (%v/%v done)",
		// 	numDone, numResources)

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
				if ready(r, g) {
					// Start a goroutine to do the actual deployment
					go deployResource(r)
				}
			} else {
				if r.State == Waiting {
					r.State = Canceled
				}
			}
		}

		// for _, r := range resources {
		// 	config.Debugf("%v", r)
		// }

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

	return nil
}

// deployTemplate deloys the CloudFormation template using the Cloud Control API.
// A failed deployment will result in DeploymentResults.Succeeded = false.
// A non-nil error is returned when something unexpected caused a failure
// not related to actually deploying resources, like an invalid template.
func DeployTemplate(template cft.Template) (*DeploymentResults, error) {

	results := &DeploymentResults{
		Succeeded: true,
		State:     cft.Template{},
		Resources: make(map[string]*Resource),
	}

	var err error

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
		Verify that we are not deleting anything depended on by a live resource.
		Fail before deploying anything if a delete would remove a dependency.
	*/

	// Wrap Nodes in a Resource to add state
	deletes := make([]*Resource, 0)
	createsUpdates := make([]*Resource, 0)
	resourceMap := make(map[string]*Resource)
	for _, n := range nodes {
		if n.Type == "Resources" {
			y, err := getTemplateResource(template, n.Name)
			if err != nil {
				panic(fmt.Sprintf("%v not found in Resources", n.Name))
			}

			_, typeNode := s11n.GetMapValue(y, "Type")
			if typeNode == nil {
				return nil, fmt.Errorf("expected resource %v to have a Type", n.Name)
			}
			typeName := typeNode.Value

			// Determine if this is a create, update, or delete
			var action diff.ActionType
			var ident string
			var model string
			var priorJson string
			_, stateNode := s11n.GetMapValue(y, "State")
			if stateNode == nil {
				// Assume this is a new deployment
				action = diff.Create
			} else {
				config.Debugf("DeployTemplate stateNode: %v", node.ToSJson(stateNode))
				for i, s := range stateNode.Content {
					if i%2 == 0 {
						if s.Value == "Action" {
							a := stateNode.Content[i+1].Value
							action = diff.ActionType(a)
							isValid := false
							switch action {
							case diff.Create, diff.Update, diff.Delete, diff.None:
								isValid = true
							}
							if !isValid {
								return nil, fmt.Errorf("invalid Action %v for %v", a, n.Name)
							}
						} else if s.Value == "Identifier" {
							ident = stateNode.Content[i+1].Value
						} else if s.Value == "ResourceModel" {
							j := format.Jsonise(stateNode.Content[i+1])
							m, _ := json.Marshal(j)
							model = string(m)
						} else if s.Value == "PriorJson" {
							priorJson = stateNode.Content[i+1].Value
						} else {
							config.Debugf("Unexpected State key %v", s.Value)
						}
					}
				}
			}

			r := NewResource(n.Name, typeName, Waiting, y)
			r.Action = action
			r.Identifier = ident
			r.Model = model         // This will get overwritten. Do we need it here?
			r.PriorJson = priorJson // We need this for ccapi update

			config.Debugf("deployment set r.Model to %v", r.Model)
			// TODO: This is blank when we update

			if r.Action == diff.Delete {
				deletes = append(deletes, r)
			} else {
				createsUpdates = append(createsUpdates, r)
			}
			resourceMap[r.Name] = r
		}
	}

	// Check to make sure there are no deletes with dependents that are not being deleted
	if err = verifyDeletes(deletes, &g, resourceMap); err != nil {
		return nil, fmt.Errorf("unable to deploy, deleted resources have one or more dependents: %v", err)
	}

	// Check to make sure there are no circular dependencies
	// TODO - Does the graph do this for us already?

	// Delete everything that needs to be deleted first
	err = deployResources(deletes, results, &g)
	if err != nil {
		return nil, err
	}
	if !results.Succeeded {
		spinner.StopTimer()
		for _, resource := range results.Resources {
			fmt.Printf("%v\n", resource)
		}
		return nil, errors.New("unable to delete resources")
	}

	// Deploy the rest of the resources
	err = deployResources(createsUpdates, results, &g)
	if err != nil {
		return nil, err
	}

	return results, nil

}
