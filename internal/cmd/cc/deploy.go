package cc

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/cmd/forecast"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/dc"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/cobra"
)

// PackageTemplate reads the template and performs any necessary packaging on it
// before deployment. The rain bucket will be created if it does not already exist.
func PackageTemplate(fn string, yes bool) cft.Template {

	t, err := pkg.File(fn)
	if err != nil {
		panic(ui.Errorf(err, "error packaging template '%s'", fn))
	}

	return t
}

func deploy(cmd *cobra.Command, args []string) {
	fn := args[0]
	name := args[1]
	base := filepath.Base(fn)
	absPath, _ := filepath.Abs(fn)

	if !Experimental {
		panic("Please add the --experimental arg to use this feature")
	}

	// Call RainBucket for side-effects in case we want to force bucket creation
	// TODO: Use the 'yes' arg instead of true
	bucketName := s3.RainBucket(true)

	// Package template
	spinner.Push(fmt.Sprintf("Preparing template '%s'", base))
	template := PackageTemplate(fn, true)
	spinner.Pop()

	// Get parameters and tags
	stack := types.Stack{} // Not relevant here
	stack.Parameters = make([]types.Parameter, 0)
	dc, err := dc.GetDeployConfig(tags, params, configFilePath, base,
		template, stack, false, yes, ignoreUnknownParams)
	if err != nil {
		panic(err)
	}
	templateConfig = dc

	// Before we do anything else, make sure that all types in the template
	// are fully supported by Cloud Control API
	types, err := template.GetTypes()
	if err != nil {
		panic(err)
	}
	config.Debugf("types: %v", types)
	anyUnsupported := false
	for _, typ := range types {
		supported, err := cfn.IsCCAPI(typ)
		if err != nil {
			panic(err)
		}
		if !supported {
			anyUnsupported = true
			fmt.Println(console.Red(fmt.Sprintf("%s is not full supported by CCAPI", typ)))
		}
	}
	if anyUnsupported {
		panic("Unable to deploy this template due to unsupported resources")
	}

	// TODO - Check for drift somewhere.. maybe in checkState..
	// Go through the UI in drift before locking the state file

	// Compare against the current state to see what has changed, if this is an update
	spinner.Push("Checking state")
	stateResult, stateError := checkState(name, template, bucketName, "", absPath, unlock)
	if stateError != nil {
		panic(stateError)
	}
	spinner.Pop()

	config.Debugf("StateFile:\n%v", format.String(stateResult.StateFile,
		format.Options{JSON: false, Unsorted: false}))

	var changes cft.Template

	if stateResult.IsUpdate {
		var err error
		changes, err = update(stateResult.StateFile, template)
		if err != nil {
			panic(err)
		}
		// TODO: update needs to take parameters into account

	} else {
		// Deploy the provided template for the first time
		changes = template
	}

	summarizeChanges(changes)

	if !console.Confirm(true, "Do you wish to continue?") {
		// Unlock the state file
		if !stateResult.IsUpdate {
			err := deleteState(name, bucketName)
			if err != nil {
				panic(fmt.Errorf("unable to remove state file: %v", err))
			}
		} else {
			err := writeState(template, nil, bucketName, name, absPath)
			if err != nil {
				panic(fmt.Errorf("unable to unlock state file: %v", err))
			}
		}

		// Exit
		panic(errors.New("user cancelled deployment"))
	}

	// Set the global reference that anything in this package can access
	deployedTemplate = changes

	// Figure out how long we thing the stack will take to execute
	totalSeconds := forecast.PredictTotalEstimate(changes, stateResult.IsUpdate)
	// TODO: Forecast can be more accurate here since we know the actions
	fmt.Printf("Predicted deployment time: %v\n", forecast.FormatEstimate(totalSeconds))

	spinner.StartTimer(fmt.Sprintf("Deploying %v", name))
	results, err := DeployTemplate(changes)
	if err != nil {
		// An unexpected error that prevented deployment from starting
		panic(err)
	}
	spinner.StopTimer()

	results.Summarize()

	if !results.Succeeded {
		panic("Deployment failed! The state file is locked and will need to be resolved manually.")

		// Leave the state file locked. Needs to be resolved manually.
	} else {
		fmt.Println("Deployment completed successfully!")

		// Unlock the state file and record current values
		err := writeState(template, results, bucketName, name, absPath)
		if err != nil {
			panic(fmt.Errorf("unable to write state file: %v", err))
		}
	}

}

var CCDeployCmd = &cobra.Command{
	Use:   "deploy <template> <name>",
	Short: "Deploy a local template directly using the Cloud Control API (Experimental!)",
	Long: `Creates or updates resources directly using Cloud Control API from the template file <template>.
You must pass the --experimental (-x) flag to use this command, to acknowledge that it is experimental and likely to be unstable!
`,
	Args:                  cobra.ExactArgs(2),
	DisableFlagsInUseLine: true,
	Run:                   deploy,
}

func init() {
	CCDeployCmd.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
	CCDeployCmd.Flags().BoolVarP(&Experimental, "experimental", "x", false, "Acknowledge that this is an experimental feature")
	CCDeployCmd.Flags().BoolVarP(&yes, "yes", "y", false, "don't ask questions; just deploy")
	CCDeployCmd.Flags().BoolVarP(&downloadState, "state", "s", false, "Instead of deploying, download the state file")
	CCDeployCmd.Flags().StringSliceVar(&tags, "tags", []string{}, "add tags to the stack; use the format key1=value1,key2=value2")
	CCDeployCmd.Flags().StringSliceVar(&params, "params", []string{}, "set parameter values; use the format key1=value1,key2=value2")
	CCDeployCmd.Flags().StringVarP(&configFilePath, "config", "c", "", "YAML or JSON file to set tags and parameters")
	CCDeployCmd.Flags().StringVarP(&unlock, "unlock", "u", "", "Unlock <lockid> and continue")
	CCDeployCmd.Flags().BoolVarP(&ignoreUnknownParams, "ignore-unknown-params", "", false, "Ignore unknown parameters")

	resMap = make(map[string]*Resource)

}
