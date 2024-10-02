package cc

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const FILE_PATH string = "FilePath"

type StateResult struct {
	StateFile cft.Template
	Lock      string
	IsUpdate  bool
}

// addCommon adds common elements to the state file
// If the elements already exist, they are replaced
func addCommon(stateMap *yaml.Node, absPath string) {
	// Record the absolute path (helps figure out who/where the template came from)
	if _, fp, _ := s11n.GetMapValue(stateMap, FILE_PATH); fp != nil {
		config.Debugf("addCommon overwriting")
		fp.Value = absPath
	} else {
		config.Debugf("addCommon adding")
		node.Add(stateMap, FILE_PATH, absPath)
	}
}

// deleteState removes the state file.
// This is necessary when the user cancels a fresh deployment
func deleteState(name string, bucketName string) error {
	key := getStateFileKey(name)
	return s3.DeleteObject(bucketName, key, nil)
}

// Get the object key for the state file in S3
func getStateFileKey(name string) string {
	key := fmt.Sprintf("deployments/%v.yaml", name)
	if s3.BucketKeyPrefix != "" {
		key = fmt.Sprintf("%s/%s", s3.BucketKeyPrefix, key)
	}
	return key
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
	priorLock string,
	absPath string,
	unlockId string) (*StateResult, error) {

	spinner.Push("Checking state")

	key := getStateFileKey(name)
	var state cft.Template

	result := &StateResult{}

	// TODO: Handle race conditions (which seem unlikely but who knows...)
	// Double check before writing a lock that someone else didn't grab it?
	// Want to avoid using another service like DDB for this. Keep it simple.

	obj, err := s3.GetObject(bucketName, key)
	spinner.Pop()
	if err != nil {

		// Make sure it's a NotFound error
		var nf *types.NoSuchKey
		if !errors.As(err, &nf) {
			return nil, err
		}

		config.Debugf("No state file found, creating")

		// This is a create operation. Create a state file and lock it.
		spinner.Push("Creating a new state file")
		lock := uuid.New().String()
		config.Debugf("Creating new state file with lock %v", lock)
		state = cft.Template{Node: node.Clone(template.Node)}
		result.StateFile = state
		result.Lock = lock
		result.IsUpdate = false

		// Edit the state template to add a new top level "State" section
		stateMap := cft.AppendStateMap(state)

		// Lock it
		node.Add(stateMap, "Lock", lock)

		// Add common elements
		addCommon(stateMap, absPath)

		// Write the state file to the bucket
		str := format.String(state, format.Options{JSON: false, Unsorted: false})
		err := s3.PutObject(bucketName, key, []byte(str))
		spinner.Pop()
		if err != nil {
			return nil, fmt.Errorf("unable to write state to bucket: %v", err)
		}

		config.Debugf("State file created with lock: %v", lock)

	} else {
		// The state file exists. Inspect it to see if it's locked

		config.Debugf("Found existing state file")

		state, err := parse.String(string(obj))
		if err != nil {
			return nil, fmt.Errorf("unable to parse state file: %v", err)
		}

		_, stateMap, _ := s11n.GetMapValue(state.Node.Content[0], "State")
		if stateMap == nil {
			return nil, fmt.Errorf("did not find State in state file")
		}

		result.StateFile = state
		result.IsUpdate = true
		lock := ""
		for i, s := range stateMap.Content {
			if s.Kind == yaml.ScalarNode && s.Value == "Lock" {
				lock = stateMap.Content[i+1].Value
			}
		}
		result.Lock = lock

		ignoreLock := false
		if lock != "" {
			if unlockId != "" {
				if lock != unlockId {
					return nil, fmt.Errorf("unlock %v does not match found lock %v", unlockId, lock)
				} else {
					fmt.Println("Unlocking the locked state file")
					ignoreLock = true
				}
			} else {
				msg := fmt.Sprintf("Found a locked state file (lock: %v). This means another process is currently deploying this template, or a deployment failed to complete. You will need to manually resolve the issue, or you can try to resume the deployment by running cc deploy with --unlock <lock>", lock)
				return nil, errors.New(msg)
			}
		}

		// Check to see if the deployment has drifted
		if err := runDriftOnState(name, state, bucketName, key); err != nil {
			return nil, err
		}

		// We are safe to proceed with an update.
		// Write a new lock back to the state file stored in S3.
		// If we're unlocking to continue a failed deployment, leave it
		// since we would need to re-lock it here anyway
		if !ignoreLock {
			lock = uuid.New().String()
			node.Add(stateMap, "Lock", lock)
		}

		// Add common elements
		addCommon(stateMap, absPath)

		str := format.String(state, format.Options{JSON: false, Unsorted: false})
		err = s3.PutObject(bucketName, key, []byte(str))
		if err != nil {
			return nil, fmt.Errorf("unable to write updated state file to bucket: %v", err)
		}
		config.Debugf("State file updated with lock: %v", lock)
	}

	return result, nil
}

// writeState writes updated state to the state file in S3 and unlocks it
// The state passed in should be the original template, since we will
// overwrite state with current values.
func writeState(
	state cft.Template,
	results *DeploymentResults,
	bucketName string,
	name string,
	absPath string) error {

	original := format.String(state, format.Options{JSON: false, Unsorted: false})
	config.Debugf("writeState original template: %v", original)

	if results != nil {
		stateMap := cft.AppendStateMap(state)
		node.Add(stateMap, "LastWriteTime", time.Now().Format(time.RFC3339))
		addCommon(stateMap, absPath)
		resourceModels := node.AddMap(stateMap, "ResourceModels")

		// Iterate over each resource in the results.
		// Add a State section to the state resource and write the resource model

		rootMap := state.Node.Content[0]
		_, resourceMap, _ := s11n.GetMapValue(rootMap, "Resources")
		if resourceMap == nil {
			panic("Expected to find a Resources section in the template")
		}

		for name, resource := range results.Resources {
			if resource.Action == diff.Delete {
				config.Debugf("Resource %v was deleted, not writing to state", name)
				continue
			}
			config.Debugf("Writing %v to state file", name)
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

			resourceStateMap := node.AddMap(resourceModels, name)
			node.Add(resourceStateMap, "Identifier", resource.Identifier)
			modelMap := node.AddMap(resourceStateMap, "Model")
			var parsed map[string]any
			json.Unmarshal([]byte(resource.Model), &parsed)
			var n yaml.Node
			err := n.Encode(parsed)
			if err != nil {
				return err
			}
			modelMap.Content = append(modelMap.Content, n.Content...)
		}
	}

	str := format.String(state, format.Options{JSON: false, Unsorted: false})
	config.Debugf("About to write state file:\n%v", str)
	key := getStateFileKey(name)
	err := s3.PutObject(bucketName, key, []byte(str))
	if err != nil {
		return fmt.Errorf("unable to write unlocked state file to bucket: %v", err)
	}

	return nil
}

// run is the cobra command for rain cc state
func runState(cmd *cobra.Command, args []string) {
	name := args[0]

	if !Experimental {
		panic("Please add the --experimental arg to use this feature")
	}

	// Call RainBucket for side-effects in case we want to force bucket creation
	bucketName := s3.RainBucket(false)

	key := getStateFileKey(name)

	obj, err := s3.GetObject(bucketName, key)
	if err != nil {
		fmt.Printf("Unable to download state: %v", err)
		return
	}

	fmt.Println(string(obj))
}

var CCStateCmd = &cobra.Command{
	Use:   "state <name>",
	Short: "Download the state file for a template deployed with cc deploy",
	Long: `When deploying templates with the cc command, a state file is created and stored in the rain assets bucket. This command outputs the contents of that file.
`,
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run:                   runState,
}

func init() {
	addCommonParams(CCStateCmd)
}
