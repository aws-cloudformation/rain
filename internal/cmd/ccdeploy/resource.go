package ccdeploy

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft/diff"
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
	Name       string
	Type       string
	Node       *yaml.Node
	State      ResourceState
	Message    string
	Identifier string
	Model      string
	Action     diff.ActionType
	// TODO - Add elapsed time
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
		return fmt.Sprintf("%s %s: %s: %v", r.Type, r.Name, state, r.Message)
	} else {
		return fmt.Sprintf("%s %s: %s", r.Type, r.Name, state)
	}
}

// NewResource creates a new Resource and adds it to the map
func NewResource(name string,
	resourceType string, state ResourceState, node *yaml.Node) *Resource {

	r := &Resource{Name: name, Type: resourceType, State: state, Node: node}
	resMap[name] = r // TODO - This is global, do we really need it?
	return r
}
