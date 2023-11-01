package ccrm

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/cmd/ccdeploy"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var yes bool
var detach bool
var roleArn string
var Experimental bool

// Cmd is the rm command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "ccrm <name>",
	Short:                 "Delete a deployment created by ccdeploy (Experimental!)",
	Long:                  "Deletes the resources in the ccdeploy deployment named <name> and waits for all CloudControl API calls to complete. This is an experimental feature that requires the -x flag to run.",
	Args:                  cobra.ExactArgs(1),
	Aliases:               []string{"ccremove", "ccdel", "ccdelete"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		spinner.Push("Fetching deployment status")
		key := fmt.Sprintf("%v/%v.yaml", ccdeploy.STATE_DIR, name) // deployments/name
		var state cft.Template

		// Call RainBucket for side-effects in case we want to force bucket creation
		bucketName := s3.RainBucket(true)

		obj, err := s3.GetObject(bucketName, key)
		if err != nil {
			panic(err)
		}

		state, err = parse.String(string(obj))
		if err != nil {
			panic(fmt.Errorf("unable to parse state file: %v", err))
		}

		_, stateMap := s11n.GetMapValue(state.Node.Content[0], "State")
		if stateMap == nil {
			panic(fmt.Errorf("did not find State in state file"))
		}

		lock := ""
		for i, s := range stateMap.Content {
			if s.Kind == yaml.ScalarNode && s.Value == "Lock" {
				lock = stateMap.Content[i+1].Value
			}
		}

		spinner.Pop()

		if lock != "" {
			msg := "Unable to remove deployment, found a locked state file"
			panic(fmt.Errorf("%v:\ns3://%v/%v (%v)", msg, bucketName, key, lock))
		}

		if !yes {
			if !console.Confirm(false, "Are you sure you want to delete this deployment?") {
				//lint:ignore ST1005 NA
				panic(fmt.Errorf("Deployment removal cancelled: '%s'", name))
			}
		}

		spinner.StartTimer(fmt.Sprintf("Removing deployment %v", name))

		// Mark each resource with the delete action
		template := cft.Template{Node: node.Clone(state.Node)}
		rootMap := template.Node.Content[0]

		_, stateResourceModels := s11n.GetMapValue(stateMap, "ResourceModels")
		if stateResourceModels == nil {
			panic("Expected to find State.ResourceModels in the state template")
		}
		identifiers := make(map[string]string, 0)
		for i, v := range stateResourceModels.Content {
			if i%2 == 0 {
				_, identifier := s11n.GetMapValue(stateResourceModels.Content[i+1], "Identifier")
				if identifier != nil {
					identifiers[v.Value] = identifier.Value
				}
			}
		}
		config.Debugf("identifiers: %v", identifiers)

		_, resourceMap := s11n.GetMapValue(rootMap, "Resources")
		for i, resource := range resourceMap.Content {
			if i%2 == 0 {
				if identifier, ok := identifiers[resource.Value]; !ok {
					panic(fmt.Errorf("unable to find identifier for %v", resource.Value))
				} else {
					r := resourceMap.Content[i+1]
					s := node.AddMap(r, "State")
					node.Add(s, "Action", "Delete")
					node.Add(s, "Identifier", identifier)
				}
			}
		}

		config.Debugf("About to delete deployment: %v", format.CftToYaml(template))

		results, err := ccdeploy.DeployTemplate(template)
		if err != nil {
			panic(err)
		}

		spinner.StopTimer()

		for _, resource := range results.Resources {
			fmt.Printf("Removed %v\n", resource)
		}
		fmt.Printf("Deployment %v successfully removed\n", name)

		spinner.Push("Deleting state file")
		err = s3.DeleteObject(bucketName, key)
		if err != nil {
			//lint:ignore ST1005 NA
			panic(fmt.Errorf("Unable to delete state file %v/%v: %v", bucketName, key, err))
		}
		spinner.Pop()
	},
}

func init() {
	Cmd.Flags().BoolVarP(&detach, "detach", "d", false, "once removal has started, don't wait around for it to finish")
	Cmd.Flags().BoolVarP(&yes, "yes", "y", false, "don't ask questions; just delete")
	Cmd.Flags().StringVar(&roleArn, "role-arn", "", "ARN of an IAM role that CloudFormation should assume to remove the stack")
	Cmd.Flags().BoolVarP(&Experimental, "experimental", "x", false, "Acknowledge that this is an experimental feature")
}
