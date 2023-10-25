package ccdeploy

import (
	"fmt"
	"path/filepath"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/spf13/cobra"
)

var params []string
var tags []string
var configFilePath string
var Experimental bool
var template cft.Template
var resMap map[string]*Resource

// PackageTemplate reads the template and performs any necessary packaging on it
// before deployment. The rain bucket will be created if it does not already exist.
func PackageTemplate(fn string, yes bool) cft.Template {

	t, err := pkg.File(fn)
	if err != nil {
		panic(ui.Errorf(err, "error packaging template '%s'", fn))
	}

	return t
}

func run(cmd *cobra.Command, args []string) {
	fn := args[0]
	name := args[1]
	base := filepath.Base(fn)

	// Call RainBucket for side-effects in case we want to force bucket creation
	bucketName := s3.RainBucket(true)

	// Package template
	spinner.Push(fmt.Sprintf("Preparing template '%s'", base))
	template = PackageTemplate(fn, true)
	spinner.Pop()

	// TODO - Get DeployConfig (modified to remove stack references...)

	// Compare against the current state to see what has changed, if this
	// is an update
	stateResult, stateError := checkState(name, template, bucketName, "")
	if stateError != nil {
		msg := fmt.Sprintf("Found a locked state file (%v). This means another process is currently deploying this template, or a deployment failed to complete. You will need to manually resolve the issue, or you can try to resume the deployment by running ccdeploy with --continue <lock>", stateError)
		panic(msg)
	}

	config.Debugf("StateFile:\n%v", format.String(stateResult.StateFile,
		format.Options{JSON: false, Unsorted: false}))

	var changes cft.Template

	if stateResult.IsUpdate {
		var err error
		changes, err = update(stateResult.StateFile, template)
		if err != nil {
			panic(err)
		}
		// Stop here for now
		// TODO - remove this
		return

	} else {
		// Deploy the provided template for the first time
		changes = template
	}

	// TODO - Resolve intrinsics (yikes!)

	results, err := deployTemplate(changes)
	if err != nil {
		// An unexpected error that prevented deployment from starting
		panic(err)
	}

	if !results.Succeeded {
		fmt.Println("Deployment failed.")

		// Leave the state file locked. Needs to be resolved manually.
	} else {
		fmt.Println("Deployment completed successfully!")

		// Unlock the state file and record current values
		writeState(template, results, bucketName, name)
	}

	for _, resource := range results.Resources {
		fmt.Printf("%v\n", resource)
	}

}

var Cmd = &cobra.Command{
	Use:   "ccdeploy <template> <name>",
	Short: "Deploy a local template directly using the Cloud Control API (Experimental!)",
	Long: `Creates or updates resources directly using Cloud Control API from the template file <template>.
You must pass the --experimental (-x) flag to use this command, to acknowledge that it is experimental and likely to be unstable!
`,
	Args:                  cobra.ExactArgs(2),
	DisableFlagsInUseLine: true,
	Run:                   run,
}

func init() {

	Cmd.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
	Cmd.Flags().StringSliceVar(&tags, "tags", []string{}, "add tags to the stack; use the format key1=value1,key2=value2")
	Cmd.Flags().StringSliceVar(&params, "params", []string{}, "set parameter values; use the format key1=value1,key2=value2")
	Cmd.Flags().StringVarP(&configFilePath, "config", "c", "", "YAML or JSON file to set tags and parameters")
	Cmd.Flags().BoolVarP(&Experimental, "experimental", "x", false, "Acknowledge that this is an experimental feature")

	resMap = make(map[string]*Resource)

}
