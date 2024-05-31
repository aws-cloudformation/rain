package stackset

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/cmd/deploy"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/dc"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
	"golang.org/x/exp/constraints"
	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
)

type configFormat struct {
	Parameters        map[string]string           `yaml:"Parameters"`
	Tags              map[string]string           `yaml:"Tags"`
	StackSet          cfn.StackSetConfig          `yaml:"StackSet"`
	StackSetInstanses cfn.StackSetInstancesConfig `yaml:"StackSetInstanses"`
}

var accounts []string
var regions []string
var detach bool
var yes bool
var params []string
var tags []string
var configFilePath string
var forceUpdate bool
var ignoreStackInstances bool

// StackSetDeployCmd is the deploy command's entrypoint
var StackSetDeployCmd = &cobra.Command{
	Use:   "deploy <template> [stackset] [flags]",
	Short: "Deploy a CloudFormation stack set from a local template",
	Long: `Creates or updates a CloudFormation stack set <stackset> from the template file <template>.
If you don't specify a stack set name, rain will use the template filename minus its extension.
If you do not specify a template file, rain will asume that you want to add a new instance to an existing template,
If a template needs to be packaged before it can be deployed, rain will package the template first.
Rain will attempt to create an S3 bucket to store artifacts that it packages and deploys.
The bucket's name will be of the format rain-artifacts-<AWS account id>-<AWS region>.

The config flags can be used to set accounts, regions to operate and tags with parameters to use.
Configuration file with extended options can be provided along with '--config' flag in YAML or JSON format (see example file for details).

YAML:
Parameters:
	Name: Value
Tags:
	Name: Value
StackSet:
	description: "test description"
	...
StackSetInstanses:
	accounts:
		- "123456789123"
	regions:
		- us-east-1
		- us-east-2
...

Account(s) and region(s) provideed as flags OVERRIDE values from configuration files. Tags and parameters from the configuration file are MERGED with CLI flag values. 
`,
	Args:                  cobra.RangeArgs(1, 2),
	DisableFlagsInUseLine: false,
	Run: func(cmd *cobra.Command, args []string) {

		templateFilePath := args[0]

		stackSetName := createStackSetName(args)

		// Convert cli flags to maps
		cliTagFlags := dc.ListToMap("tag", tags)
		cliParamFlags := dc.ListToMap("param", params)

		// Read configuration data from a file
		configData := readConfiguration(configFilePath)

		//ovveride config data with CLI flag values
		combineConfigDataWithCliFlags(&configData, cliParamFlags, cliTagFlags, accounts, regions)

		// Get current stack set if exist
		spinner.Push(fmt.Sprintf("Checking current status of stack set '%s'", stackSetName))
		existingStackSet, err := cfn.GetStackSet(stackSetName)
		spinner.Pop()
		isStacksetExists := false
		if err == nil && existingStackSet.Status != types.StackSetStatusDeleted {
			isStacksetExists = true
		}
		configData.StackSet.StackSetName = stackSetName
		configData.StackSetInstanses.StackSetName = stackSetName

		// Package template, if we add new instances templay is not needed
		spinner.Push(fmt.Sprintf("Preparing template '%s'", templateFilePath))
		configData.StackSet.Template = deploy.PackageTemplate(templateFilePath, yes)
		spinner.Pop()

		// Build []types.Parameter from configuration data
		config.Debugln("Handling parameters")
		configData.StackSet.Parameters = buildParameterTypes(configData.StackSet.Template, configData.Parameters, existingStackSet)

		// Build []types.Tag from configuration data
		config.Debugln("Handling tags")
		configData.StackSet.Tags = dc.MakeTags(configData.Tags)

		if config.Debug {
			for _, param := range configData.StackSet.Parameters {
				val := ptr.ToString(param.ParameterValue)
				if ptr.ToBool(param.UsePreviousValue) {
					val = "<previous value>"
				}
				config.Debugf("  %s: %s", ptr.ToString(param.ParameterKey), val)
			}
		}

		if isStacksetExists {
			if forceUpdate || console.Confirm(true, "Stack set already exists. Do you want to update it?") {
				updateStackSet(configData)
				if !ignoreStackInstances {
					addInstances(configData)
				}

			} else {
				fmt.Println(console.Yellow("operation was cancelled by user"))
			}
		} else {
			createStackSet(configData)
		}
	},
}

func init() {
	dc.FixStackNameRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

	StackSetDeployCmd.Flags().StringSliceVar(&accounts, "accounts", []string{}, "accounts for which to create stack set instances")
	StackSetDeployCmd.Flags().StringSliceVar(&regions, "regions", []string{}, "regions where you want to create stack set instances")
	StackSetDeployCmd.Flags().BoolVarP(&detach, "detach", "d", false, "once deployment has started, don't wait around for it to finish")
	StackSetDeployCmd.Flags().StringSliceVar(&tags, "tags", []string{}, "add tags to the stack; use the format key1=value1,key2=value2")
	StackSetDeployCmd.Flags().StringSliceVar(&params, "params", []string{}, "set parameter values; use the format key1=value1,key2=value2")
	StackSetDeployCmd.Flags().StringVarP(&configFilePath, "config", "c", "", "YAML or JSON file to set additional configuration parameters")
	StackSetDeployCmd.Flags().BoolVarP(&forceUpdate, "yes", "y", false, "update the stackset without confirmation")
	StackSetDeployCmd.Flags().BoolVarP(&ignoreStackInstances, "ignore-stack-instances", "i", false, "ignores adding or removing stack instances while updating, useful if you are managing the stack instances separately")
}

func readConfiguration(configFilePath string) configFormat {

	var configData configFormat

	// Read configuration file
	if len(configFilePath) != 0 {
		configFileContent, err := os.ReadFile(configFilePath)
		if err != nil {
			panic(ui.Errorf(err, "unable to read config file '%s'", configFilePath))
		}

		err = yaml.Unmarshal([]byte(configFileContent), &configData)
		if err != nil {
			panic(ui.Errorf(err, "unable to parse yaml in '%s'", configFilePath))
		}
	}
	return configData
}

func combineConfigDataWithCliFlags(configData *configFormat, cliParams map[string]string, cliTags map[string]string, cliAccounts []string, cliRegions []string) {
	// Merge Tags
	for k, v := range cliTags {
		if _, ok := configData.Tags[k]; ok {
			fmt.Println(console.Yellow(fmt.Sprintf("tags flag overrides tag in config file: %s", k)))
		}
		if configData.Tags == nil {
			configData.Tags = make(map[string]string)
		}
		configData.Tags[k] = v
	}

	// Merge Params
	for k, v := range cliParams {
		if _, ok := configData.Parameters[k]; ok {
			fmt.Println(console.Yellow(fmt.Sprintf("params flag overrides parameter in config file: %s", k)))
		}
		if configData.Parameters == nil {
			configData.Parameters = make(map[string]string)
		}
		configData.Parameters[k] = v
	}

	// Override  accounts with CLI values
	if len(cliAccounts) > 0 {
		configData.StackSetInstanses.Accounts = cliAccounts
	}

	// Override regions with CLI values
	if len(cliRegions) > 0 {
		configData.StackSetInstanses.Regions = cliRegions
	}
}

// builds stack set name out of the template filename or takes it from the cli args
func createStackSetName(args []string) string {
	var stackSetName string

	base := filepath.Base(args[0])
	if len(args) == 2 {
		stackSetName = args[1]
	} else {
		stackSetName = base[:len(base)-len(filepath.Ext(base))]

		// Now ensure it's a valid cfc name
		stackSetName = dc.FixStackNameRe.ReplaceAllString(stackSetName, "-")

		if len(stackSetName) > dc.MaxStackNameLength {
			stackSetName = stackSetName[:dc.MaxStackNameLength]
		}
	}
	return stackSetName
}

// Validate if we have enough configuration data to create/update stack set instances
func isInstanceConfigDataValid(c *cfn.StackSetInstancesConfig) bool {
	if c != nil &&
		c.Regions != nil && len(c.Regions) > 0 &&
		((c.Accounts != nil && len(c.Accounts) > 0) ||
			(c.DeploymentTargets != nil && c.DeploymentTargets.OrganizationalUnitIds != nil && len(c.DeploymentTargets.OrganizationalUnitIds) > 0)) {
		config.Debugf("ConfigData is valid\n")
		return true
	} else {
		config.Debugf("ConfigData NOT valid\n")
		return false
	}
}

// removes non-existing instances from the StackSetInstancesConfig.
func removeNonExistingInstances(c *cfn.StackSetInstancesConfig) {
	// Get current stack set instances
	instances, err := cfn.ListStackSetInstances(c.StackSetName, delegatedAdmin)
	if err != nil {
		panic(ui.Errorf(err, "unable to fetch instances for stack set - '%s'", c.StackSetName))
	}
	var existingAccounts []string
	var existingRegions []string

	for _, instance := range instances {
		existingAccounts = append(existingAccounts, *instance.Account)
		existingRegions = append(existingRegions, *instance.Region)
	}

	c.Accounts = intersection(c.Accounts, existingAccounts)
	c.Regions = intersection(c.Regions, existingRegions)
}

// removes existing instances from the StackSetInstancesConfig.
// We do not remove accounts because we accept list of provided
// accounts and regions as requirement to have instances in all provided
// accounts whether updated or created(added)
func removeExistingInstances(c *cfn.StackSetInstancesConfig) {
	// Get current stack set instances
	instances, err := cfn.ListStackSetInstances(c.StackSetName, delegatedAdmin)
	if err != nil {
		panic(ui.Errorf(err, "unable to fetch instances for stack set - '%s'", c.StackSetName))
	}

	var existingRegions []string

	for _, instance := range instances {
		existingRegions = append(existingRegions, *instance.Region)
	}
	c.Regions = difference(c.Regions, existingRegions)
}

// difference returns the elements in `a` that aren't in `b`.
func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

// returns two slices intersection
func intersection[T constraints.Ordered](pS ...[]T) []T {
	hash := make(map[T]*int) // value, counter
	result := make([]T, 0)
	for _, slice := range pS {
		duplicationHash := make(map[T]bool) // duplication checking for individual slice
		for _, value := range slice {
			if _, isDup := duplicationHash[value]; !isDup { // is not duplicated in slice
				if counter := hash[value]; counter != nil { // is found in hash counter map
					if *counter++; *counter >= len(pS) { // is found in every slice
						result = append(result, value)
					}
				} else { // not found in hash counter map
					i := 1
					hash[value] = &i
				}
				duplicationHash[value] = true
			}
		}
	}
	return result
}

// creates stack set along with stack instances
func createStackSet(configData configFormat) {
	stackSetConfig := configData.StackSet
	stackSetConfig.StackSetName = configData.StackSet.StackSetName
	stackSetConfig.Parameters = configData.StackSet.Parameters
	stackSetConfig.Tags = configData.StackSet.Tags
	config.Debugf("Stack Set Configuration: \n%s\n", format.PrettyPrint(stackSetConfig))
	stackSetConfig.Template = configData.StackSet.Template

	// Create Stack Set
	spinner.Push("Creating stack set")
	stackSetId, err := cfn.CreateStackSet(stackSetConfig)
	spinner.Pop()
	if err != nil || stackSetId == nil {
		panic(ui.Errorf(err, "error while creating stack set '%s' ", configData.StackSet.StackSetName))
	} else {
		fmt.Printf("Stack set has been created successfuly with ID: %s\n", *stackSetId)
	}

	// we create instances only if there is enough configuration data was provided in a config file or as cli arguments
	if isInstanceConfigDataValid(&configData.StackSetInstanses) {
		stackSetInstancesConfig := configData.StackSetInstanses
		stackSetInstancesConfig.StackSetName = configData.StackSet.StackSetName
		stackSetInstancesConfig.CallAs = configData.StackSet.CallAs

		config.Debugf("Stack Set Instances Configuration: \n%s\n", format.PrettyPrint(stackSetInstancesConfig))

		// Create Stack Set instances
		spinner.Push("Creating stack set instances")
		err = cfn.CreateStackSetInstances(stackSetInstancesConfig, !detach)
		spinner.Pop()
		if err != nil {
			panic(ui.Errorf(err, "error while creating stack set instances"))
		}
		if !detach {
			fmt.Println("Stack set instances have been created successfully")
		} else {
			fmt.Println("Stack set instances creation was initiated successfuly")
		}
	} else {
		fmt.Println("Not enough information provided to create stack set instance(s). Please use configuration file or provide account(s) and region(s) for deployment as command argiments")
	}
}

// converts 'string' parameters to typed objects
func buildParameterTypes(template cft.Template, combinedParams map[string]string, stackSet *types.StackSet) []types.Parameter {

	defer func() { //catch or finally
		if err := recover(); err != nil { //catch
			panic(fmt.Errorf("error occured while handling parameters: %v", err))
		}
	}()

	var oldParams []types.Parameter
	var stackSetExist = false
	if stackSet != nil {
		oldParams = stackSet.Parameters
		stackSetExist = true
	}
	return dc.GetParameters(template, combinedParams, oldParams, stackSetExist, true, false)
}

// updates existing stack set and all its instances
func updateStackSet(configData configFormat) {
	config.Debugf("Updating Stack Set: %s\nStack Set Configuration: \n%s\nStack Set Instances Configuration: \n%s\n",
		configData.StackSet.StackSetName, format.PrettyPrint(configData.StackSet), format.PrettyPrint(configData.StackSetInstanses))

	// remove accounts and regions for the instances that do not exist, removed instances supposed to be created but not updated
	if !ignoreStackInstances {
		removeNonExistingInstances(&configData.StackSetInstanses)
	}

	// check if we have instances left to update after filtering
	if !ignoreStackInstances && !isInstanceConfigDataValid(&configData.StackSetInstanses) {
		fmt.Println("There is no instances to update.")
		return
	}

	// Update Stack Set with its instances
	spinner.Push("Updating stack set")

	// making a copy to avoid mutating the global configuration
	stackSetInstances := configData.StackSetInstanses
	if ignoreStackInstances {
		stackSetInstances.Accounts = nil
		stackSetInstances.Regions = nil
	}
	err := cfn.UpdateStackSet(configData.StackSet, stackSetInstances, !detach)
	spinner.Pop()
	if err != nil {
		panic(ui.Errorf(err, "error occurred while updating stack set '%s' ", configData.StackSetInstanses.StackSetName))
	} else {
		fmt.Println("Stack set update has been completed.")
	}

}

// adds stack set instances to an existing stack set
func addInstances(configData configFormat) {
	config.Debugf("Adding Stack Set instance(s): %s\nStack Set Configuration: \n%s\nStack Set Instances Configuration: \n%s\n",
		configData.StackSet.StackSetName, format.PrettyPrint(configData.StackSet), format.PrettyPrint(configData.StackSetInstanses))

	// remove existing instances from configData
	removeExistingInstances(&configData.StackSetInstanses)

	// check if we have accounts and regions to update
	if !isInstanceConfigDataValid(&configData.StackSetInstanses) {
		fmt.Println("There are no new instances to be created.")
		os.Exit(0)
	}

	spinner.Push("Adding stack set instances")
	err := cfn.AddStackSetInstances(configData.StackSet, configData.StackSetInstanses, !detach)
	spinner.Pop()
	if err != nil {
		panic(ui.Errorf(err, "error occurred while adding stack set instances for stack set'%s' ", configData.StackSet.StackSetName))
	} else {
		fmt.Println("Stack set update has been completed.")
	}
}
