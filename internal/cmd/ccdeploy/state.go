package ccdeploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

const STATE_DIR string = "deployments"

type StateResult struct {
	StateFile cft.Template
	Lock      string
	IsUpdate  bool
}

// checkState looks for an existing state file.
//
// If one does not exist, it is created.
//
// If one exists and there is a lock, an error is returned unless this process
// owns the lock.  If priorLock matches, then we are in the middle of an update
// and for some reason we needed to re-check state
//
// If one exists and there is no lock, this is an update.
// Save the state file back with a lock that we own
func checkState(
	name string,
	template cft.Template,
	bucketName string,
	priorLock string) (*StateResult, error) {

	spinner.Push("Checking state")

	key := fmt.Sprintf("%v/%v.yaml", STATE_DIR, name) // deployments/name
	var state cft.Template

	result := &StateResult{}

	// TODO: Handle race conditions (which seem unlikely but who knows...)
	// Double check before writing a lock that someone else didn't grab it?
	// Want to avoid using another service like DDB for this. Keep it simple.

	obj, err := s3.GetObject(bucketName, key)
	if err != nil {
		config.Debugf("checkState GetObject: %v", err)

		// Make sure it's a NotFound error
		var nf *types.NoSuchKey
		if !errors.As(err, &nf) {
			return nil, err
		}

		spinner.Push("No state file found, creating")

		// This is a create operation. Create a state file and lock it.
		lock := uuid.New().String()
		config.Debugf("Creating new state file with lock %v", lock)
		state = cft.Template{Node: node.Clone(template.Node)}
		result.StateFile = state
		result.Lock = lock
		result.IsUpdate = false

		// Edit the state template to add a new top level "State" section
		stateMap := appendStateMap(state)

		// Lock it
		add(stateMap, "Lock", lock)

		// Write the state file to the bucket
		str := format.String(state, format.Options{JSON: false, Unsorted: false})
		err := s3.PutObject(bucketName, key, []byte(str))
		if err != nil {
			return nil, fmt.Errorf("unable to write state to bucket: %v", err)
		}

		spinner.Push(fmt.Sprintf("State file created with lock: %v", lock))

	} else {
		// The state file exists. Inspect it to see if it's locked

		config.Debugf("checkState state file exists")

		spinner.Push("Found existing state file")

		state, err := parse.String(string(obj))
		if err != nil {
			return nil, fmt.Errorf("unable to parse state file: %v", err)
		}

		config.Debugf("state:\n%v", node.ToSJson(state.Node))

		_, stateMap := s11n.GetMapValue(state.Node.Content[0], "State")
		if stateMap == nil {
			return nil, fmt.Errorf("did not find State in state file")
		}

		config.Debugf("stateMap:\n%v", node.ToSJson(stateMap))

		result.StateFile = state
		result.IsUpdate = true
		lock := ""
		for i, s := range stateMap.Content {
			if s.Kind == yaml.ScalarNode && s.Value == "Lock" {
				lock = stateMap.Content[i+1].Value
			}
		}
		result.Lock = lock

		if lock != "" {
			return nil, fmt.Errorf("lock: %v", lock)
		}

		// We are safe to proceed with an update.
		// Write a new lock back to the state file stored in S3.
		lock = uuid.New().String()
		add(stateMap, "Lock", lock)
		str := format.String(state, format.Options{JSON: false, Unsorted: false})
		err = s3.PutObject(bucketName, key, []byte(str))
		if err != nil {
			return nil, fmt.Errorf("unable to write updated state file to bucket: %v", err)
		}
		spinner.Push(fmt.Sprintf("State file updated with lock: %v", lock))
	}

	return result, nil
}

// appendStateMap appends a "State" section to the template
func appendStateMap(state cft.Template) *yaml.Node {
	state.Node.Content[0].Content = append(state.Node.Content[0].Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "State"})
	stateMap := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
	state.Node.Content[0].Content = append(state.Node.Content[0].Content, stateMap)
	return stateMap
}

// add adds a new property to the state map
func add(stateMap *yaml.Node, name string, val string) {
	stateMap.Content = append(stateMap.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: name})
	stateMap.Content = append(stateMap.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: val})
}

// addMap adds a new map to the parent node
func addMap(parent *yaml.Node, name string) *yaml.Node {
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: name})
	m := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
	parent.Content = append(parent.Content, m)
	return m
}

// writeState writes updated state to the state file in S3 and unlocks it
// The state passed in should be the original template, since we will
// overwrite state with current values.
func writeState(
	state cft.Template,
	results *DeploymentResults,
	bucketName string,
	name string) error {

	stateMap := appendStateMap(state)
	add(stateMap, "LastWriteTime", time.Now().Format(time.RFC3339))

	// Iterate over each resource in the results.
	// Add a State section to the state resource and write the resource model

	rootMap := state.Node.Content[0]
	_, resourceMap := s11n.GetMapValue(rootMap, "Resources")
	if resourceMap == nil {
		panic("Expected to find a Resources section in the template")
	}

	config.Debugf("resourceMap: %v", node.ToSJson(resourceMap))

	for name, resource := range results.Resources {
		config.Debugf("writeState resource %v\n%v", name, resource.Model)

		var stateResource *yaml.Node
		for i, r := range resourceMap.Content {
			if r.Value == name {
				stateResource = resourceMap.Content[i+1]
				break
			}
		}
		if stateResource == nil {
			return fmt.Errorf("did not find %v in the state template", name)
		}
		config.Debugf("stateResource: %v", node.ToSJson(stateResource))

		resourceStateMap := addMap(stateResource, "State")
		add(resourceStateMap, "Identifier", resource.Identifier)
		modelMap := addMap(resourceStateMap, "Model")
		var parsed map[string]any
		json.Unmarshal([]byte(resource.Model), &parsed)
		var n yaml.Node
		err := n.Encode(parsed)
		if err != nil {
			return err
		}
		config.Debugf("ResourceModel node: %v", node.ToSJson(&n))
		for _, c := range n.Content {
			modelMap.Content = append(modelMap.Content, c)
		}

		config.Debugf("ResourceModel map: %v", node.ToSJson(modelMap))
		config.Debugf("stateResource: %v", node.ToSJson(stateResource))
	}

	str := format.String(state, format.Options{JSON: false, Unsorted: false})
	config.Debugf("About to write state file:\n%v", str)
	key := fmt.Sprintf("%v/%v.yaml", STATE_DIR, name) // deployments/name
	err := s3.PutObject(bucketName, key, []byte(str))
	if err != nil {
		return fmt.Errorf("unable to write unlocked state file to bucket: %v", err)
	}
	spinner.Push(fmt.Sprintf("State file written and unlocked"))

	return nil
}
