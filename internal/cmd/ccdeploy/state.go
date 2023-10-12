package ccdeploy

import (
	"errors"
	"fmt"

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
// If one exists and there is no lock, this is an update
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
		state.Node.Content[0].Content = append(state.Node.Content[0].Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "State"})
		stateMap := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
		state.Node.Content[0].Content = append(state.Node.Content[0].Content, stateMap)
		stateMap.Content = append(stateMap.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "Lock"})
		stateMap.Content = append(stateMap.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: lock})

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

		// Otherwise we are safe to proceed with an update

	}

	return result, nil
}
