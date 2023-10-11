package ccdeploy

import (
	"fmt"
	"path/filepath"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/node"
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
	stateTemplate, stateError := checkState(name, template, bucketName)
	if stateError != nil {
		panic(stateError)
	}

	config.Debugf("stateTemplate:\n%v", node.ToSJson(stateTemplate.Node))

	// Create a diff between the current state and template
	// TODO

	results := deployTemplate(template)

	if !results.Succeeded {
		fmt.Println("Deployment failed.")
		// TODO - Error message?
		// TODO - Instructions on what to do now?
	} else {
		fmt.Println("Deployment completed successfully!")
	}

	for name, resource := range results.Resources {
		fmt.Printf("%v: %v\n", name, resource)
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
