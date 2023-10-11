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

type ResourceState int

const (
	Waiting ResourceState = iota
	Deploying
	Failed
	Deployed
	Canceled
)

type Resource struct {
	Name    string
	Node    *yaml.Node
	State   ResourceState
	Message string
}

func (r Resource) String() string {
	state := ""
	switch r.State {
	case Waiting:
		state = "Waiting"
	case Deploying:
		state = "Deploying"
	case Deployed:
		state = "Deployed"
	case Failed:
		state = "Failed"
	case Canceled:
		state = "Canceled"
	}
	if r.State == Failed {
		return fmt.Sprintf("%s: %s: %v", r.Name, state, r.Message)
	} else {
		return fmt.Sprintf("%s: %s", r.Name, state)
	}
}

// NewResource creates a new Resource and adds it to the map
func NewResource(name string, state ResourceState, node *yaml.Node) *Resource {
	r := &Resource{Name: name, State: state, Node: node}
	resMap[name] = r
	return r
}

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
	config.Debugf("Simulate deploying %v...", resource)

	resource.State = Deploying

	// Get the properties and call ccapi
	err := ccapi.CreateResource(resource.Name, resource.Node)
	if err != nil {
		config.Debugf("deployResource failed: %v", err)
		resource.State = Failed
		resource.Message = fmt.Sprintf("%v", err)
	} else {
		resource.State = Deployed
		resource.Message = "Success"
	}
}

// ready returns true if the resource has no undeployed dependencies
func ready(resource *Resource, g *graph.Graph) bool {

	// Iterate over each of this resource's dependencies
	for _, dep := range g.Get(graph.Node{Name: resource.Name, Type: "Resources"}) {

		config.Debugf("ready: %v depends on %v", resource.Name, dep)

		if dep.Type != "Resources" {
			continue
		}

		depr := resMap[dep.Name]

		// If the dependency is not deployed, terminate
		if depr.State != Deployed {
			config.Debugf("ready: %v has not been deployed", depr)
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
func deployTemplate(template cft.Template) DeploymentResults {

	results := DeploymentResults{
		Succeeded: true,
		State:     cft.Template{},
		Resources: make(map[string]*Resource),
	}

	results.State.Node = node.Clone(template.Node)

	// Create a dependency graph of the template
	g := graph.New(template)
	nodes := g.Nodes()

	config.Debugf("Found %v nodes in the template", len(nodes))

	/*
		Downwards is "depends on"

		   A   E  F
		   /    \\
		 B   C      GH
			  \
			   D

		B, D, G, and H can all be deployed at the same time.

		We work our way up from the bottom, deploying resources concurrently
		as soon as they have no more undeployed dependencies.

	*/

	// Wrap Nodes in a Resource to add state
	resources := make([]*Resource, 0)
	for _, n := range nodes {
		config.Debugf("node: %v", n)
		if n.Type == "Resources" {
			y, err := getTemplateResource(n.Name)
			if err != nil {
				panic(fmt.Sprintf("%v not found", n.Name))
			}

			config.Debugf("resource Node:\n%v", node.ToSJson(y))
			r := NewResource(n.Name, Waiting, y)
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

			if r.State == Deployed {
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

	return results

}
