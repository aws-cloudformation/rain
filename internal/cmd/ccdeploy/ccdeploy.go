package ccdeploy

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/graph"
	"github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var params []string
var tags []string
var configFilePath string
var Experimental bool
var template cft.Template

type ResourceState int

const (
	Waiting ResourceState = iota
	Deploying
	Deployed
)

type Resource struct {
	Node  graph.Node
	State ResourceState
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
	}
	return fmt.Sprintf("%s/%s: %s", r.Node.Type, r.Node.Name, state)
}

// NewResource creates a new Resource and adds it to the map
func NewResource(n graph.Node, state ResourceState) *Resource {
	r := &Resource{n, state}
	resMap[n] = r
	return r
}

// PackageTemplate reads the template and performs any necessary packaging on it
// before deployment. The rain bucket will be created if it does not already exist.
// TODO - What about state management? Do we initialize that here?
func PackageTemplate(fn string, yes bool) cft.Template {
	// Call RainBucket for side-effects in case we want to force bucket creation
	s3.RainBucket(yes)

	t, err := pkg.File(fn)
	if err != nil {
		panic(ui.Errorf(err, "error packaging template '%s'", fn))
	}

	return t
}

// TODO - Set ResourceState with a mutex?

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

// deploy calls the Cloud Control API to deploy the resource
func deploy(resource *Resource) {
	config.Debugf("Simulate deploying %v...", resource)

	y, err := getTemplateResource(resource.Node.Name)
	if err != nil {
		panic(fmt.Sprintf("%v not found", resource.Node.Name))
	}

	config.Debugf("deploy:\n%v", node.ToSJson(y))

	resource.State = Deploying
	time.Sleep(time.Second * 3)
	resource.State = Deployed
}

var resMap map[graph.Node]*Resource

// ready returns true if the resource has no undeployed dependencies
func ready(resource *Resource, g *graph.Graph) bool {
	n := resource.Node

	// Iterate over each of this resource's dependencies
	for _, dep := range g.Get(n) {

		config.Debugf("ready: %v depends on %v", resource.Node.Name, dep)

		if dep.Type != "Resources" {
			continue
		}

		depr := resMap[dep]

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

func run(cmd *cobra.Command, args []string) {
	fn := args[0]
	base := filepath.Base(fn)

	// Package template
	spinner.Push(fmt.Sprintf("Preparing template '%s'", base))
	template = PackageTemplate(fn, true)
	spinner.Pop()

	// TODO - Get DeployConfig (modified to remove stack references...)

	// Compare against the current state to see what has changed, if this
	// is an update

	// Create a dependency graph of the template
	g := graph.New(template)
	nodes := g.Nodes()

	config.Debugf("Found %v nodes in the template", len(nodes))

	/*
		Downwards is "depends on"

		   A   E  F
		  / \ /    \\
		 B   C      GH
			  \
			   D

		B, D, G, and H can all be deployed at the same time.

		We work our way up from the bottom, deploying resources concurrently
		as soon as they have no more undeployed dependencies.

	*/

	// Wrap Nodes in a Resource to add fields like Deployed
	resources := make([]*Resource, 0)
	for _, n := range nodes {
		config.Debugf("node: %v", n)
		if n.Type == "Resources" {
			r := NewResource(n, Waiting)
			resources = append(resources, r)
		}
	}

	numResources := len(resources)
	numDone := 0

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

			// Recurse dependencies to see if it's ok to deploy this one now
			if ready(r, &g) {
				// Start a goroutine to do the actual deployment
				go deploy(r)
			}
		}

		for _, r := range resources {
			config.Debugf("%v", r)
		}

		// Give deployment routines time to finish
		// TODO: We could be smarter about this with channels...
		time.Sleep(time.Second * 1)
	}

	fmt.Println("Deployment complete")
}

var Cmd = &cobra.Command{
	Use:   "ccdeploy <template>",
	Short: "Deploy a local template directly using the Cloud Control API (Experimental!)",
	Long: `Creates or updates resources directly using Cloud Control API from the template file <template>.
You must pass the --experimental (-x) flag to use this command, to acknowledge that it is experimental and likely to be unstable!
`,
	Args:                  cobra.RangeArgs(1, 2),
	DisableFlagsInUseLine: true,
	Run:                   run,
}

func init() {

	Cmd.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
	Cmd.Flags().StringSliceVar(&tags, "tags", []string{}, "add tags to the stack; use the format key1=value1,key2=value2")
	Cmd.Flags().StringSliceVar(&params, "params", []string{}, "set parameter values; use the format key1=value1,key2=value2")
	Cmd.Flags().StringVarP(&configFilePath, "config", "c", "", "YAML or JSON file to set tags and parameters")
	Cmd.Flags().BoolVarP(&Experimental, "experimental", "x", false, "Acknowledge that this is an experimental feature")

	resMap = make(map[graph.Node]*Resource)

}
