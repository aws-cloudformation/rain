package ccrm

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/cmd/ccdeploy"
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
	Short:                 "Delete a deployment created by ccdeploy",
	Long:                  "Deletes the resources in the ccdeploy deployment named <name> and waits for all CloudControl API calls to complete. This is an experimental feature that requires the -x flag to run.",
	Args:                  cobra.ExactArgs(1),
	Aliases:               []string{"ccremove", "ccdel", "ccdelete"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		spinner.Push("Fetching deployment status")
		// TODO - Check for an unlocked state file
		key := fmt.Sprintf("%v/%v.yaml", ccdeploy.STATE_DIR, name) // deployments/name
		var state cft.Template

		// Call RainBucket for side-effects in case we want to force bucket creation
		bucketName := s3.RainBucket(true)

		obj, err := s3.GetObject(bucketName, key)
		if err != nil {
			panic(err)
		}

		spinner.Push("Found existing state file")

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

		if lock != "" {
			panic(fmt.Errorf("unable to remove this deployment, found a locked state file: %v", lock))
		}

		if !yes {

			spinner.Pause()

			if !console.Confirm(false, "Are you sure you want to delete this deployment?") {
				panic(fmt.Errorf("user cancelled deletion of deployment '%s'", name))
			}
			spinner.Resume()
		}

		spinner.Pop()

		// TODO - Delete the deployment

		template := cft.Template{Node: node.Clone(state.Node)}
		// TODO - Mark each resource with the delete action

		results, err := ccdeploy.DeployTemplate(template)
		if err != nil {
			panic(err)
		}

		spinner.Pop()
		for _, resource := range results.Resources {
			fmt.Printf("Removed %v\n", resource)
		}
		fmt.Printf("Deployment %v successfully removed\n", name)
	},
}

func init() {
	Cmd.Flags().BoolVarP(&detach, "detach", "d", false, "once removal has started, don't wait around for it to finish")
	Cmd.Flags().BoolVarP(&yes, "yes", "y", false, "don't ask questions; just delete")
	Cmd.Flags().StringVar(&roleArn, "role-arn", "", "ARN of an IAM role that CloudFormation should assume to remove the stack")
	Cmd.Flags().BoolVarP(&Experimental, "experimental", "x", false, "Acknowledge that this is an experimental feature")
}
