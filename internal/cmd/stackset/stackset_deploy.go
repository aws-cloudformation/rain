package stackset

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/cmd/deploy"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
)

const noChangeFoundMsg = "The submitted information didn't contain changes. Submit different information to create a change set."

type configFormat struct {
	Parameters        map[string]string           `yaml:"Parameters"`
	Tags              map[string]string           `yaml:"Tags"`
	StackSet          cfn.StackSetConfig          `yaml:"StackSet"`
	StackSetInstanses cfn.StackSetInstancesConfig `yaml:"StackSetInstanses"`
}

var detach bool
var yes bool
var params []string
var tags []string
var configFilePath string
var terminationProtection bool
var keep bool

// StackSetDeployCmd is the deploy command's entrypoint
var StackSetDeployCmd = &cobra.Command{
	Use:   "deploy <template> [stack]",
	Short: "Deploy a CloudFormation stack set from a local template",
	Long: `Creates or updates a CloudFormation stack set <stackset> from the template file <template>.
If you don't specify a stack set name, rain will use the template filename minus its extension.

If a template needs to be packaged before it can be deployed, rain will package the template first.
Rain will attempt to create an S3 bucket to store artifacts that it packages and deploys.
The bucket's name will be of the format rain-artifacts-<AWS account id>-<AWS region>.

The config flags can be used to programmatically set tags and parameters.
The format is similar to the "Template configuration file" for AWS CodePipeline just without the
'StackPolicy' key. The file can be in YAML or JSON format.

JSON:
  {
    "Parameters" : {
      "NameOfTemplateParameter" : "ValueOfParameter",
      ...
    },
    "Tags" : {
      "TagKey" : "TagValue",
      ...
    }
  }

YAML:
  Parameters:
    NameOfTemplateParameter: ValueOfParameter
    ...
  Tags:
    TagKey: TagValue
    ...
`,
	Args:                  cobra.RangeArgs(1, 2),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		templateFilePath := args[0]
		base := filepath.Base(templateFilePath)

		var stackSetName string
		if len(args) == 2 {
			stackSetName = args[1]
		} else {
			stackSetName = base[:len(base)-len(filepath.Ext(base))]

			// Now ensure it's a valid cfc name
			stackSetName = deploy.FixStackNameRe.ReplaceAllString(stackSetName, "-")

			if len(stackSetName) > deploy.MaxStackNameLength {
				stackSetName = stackSetName[:deploy.MaxStackNameLength]
			}
		}

		// Parse cli tags
		cliTagFlags := deploy.ListToMap("tag", tags)

		// Parse cli params
		cliParamFlags := deploy.ListToMap("param", params)

		configData := readConfiguration(configFilePath)

		mergeCliFlagsToConfigData(&configData, cliParamFlags, cliTagFlags)

		// Package template
		spinner.Push(fmt.Sprintf("Preparing template '%s'", base))
		template := deploy.PackageTemplate(templateFilePath, yes)
		spinner.Pop()

		// Check current stack set status
		spinner.Push(fmt.Sprintf("Checking current status of stack set '%s'", stackSetName))
		stackSet, err := cfn.GetStackSet(stackSetName)
		spinner.Pop()

		stackSetExists := false
		if err == nil && stackSet.Status != types.StackSetStatusDeleted { // TODO: implement update existing stackset
			//fmt.Println("can't create stack set. It already exists")
			stackSetExists = true
		}

		// Build types.Parameter from configuration data
		config.Debugf("Handling parameters")
		parameters := deploy.GetParameters(template, configData.Parameters, stackSet.Parameters, stackSetExists)

		if config.Debug {
			for _, param := range parameters {
				val := ptr.ToString(param.ParameterValue)
				if ptr.ToBool(param.UsePreviousValue) {
					val = "<previous value>"
				}
				config.Debugf("  %s: %s", ptr.ToString(param.ParameterKey), val)
			}
		}

		stackSetConfig := configData.StackSet
		stackSetConfig.StackSetName = &stackSetName
		stackSetConfig.Parameters = parameters
		stackSetConfig.Tags = cfn.MakeTags(configData.Tags)
		config.Debugf("Stack Set Configuration: \n%s\n", format.PrettyPrint(stackSetConfig))
		stackSetConfig.Template = template

		// Create Stack Set
		spinner.Push("Creating stack set")
		stackSetId, err := cfn.CreateStackSet(stackSetConfig)
		spinner.Pop()
		if err != nil || stackSetId == nil {
			panic(ui.Errorf(err, "error while creating stack set '%s' ", stackSetName))
		} else {
			fmt.Printf("Stack set has been created successfuly with ID: %s\n", *stackSetId)
		}

		stackSetInstancesConfig := configData.StackSetInstanses
		stackSetInstancesConfig.StackSetName = &stackSetName
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
			fmt.Println("Stack set instances have been created successfuly")
		} else {
			fmt.Println("Stack set instances creation was initiated successfuly")
		}

	},
}

func init() {
	deploy.FixStackNameRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

	StackSetDeployCmd.Flags().BoolVarP(&detach, "detach", "d", false, "once deployment has started, don't wait around for it to finish")
	// StackSetDeployCmd.Flags().BoolVarP(&yes, "yes", "y", false, "don't ask questions; just deploy")
	StackSetDeployCmd.Flags().StringSliceVar(&tags, "tags", []string{}, "add tags to the stack; use the format key1=value1,key2=value2")
	StackSetDeployCmd.Flags().StringSliceVar(&params, "params", []string{}, "set parameter values; use the format key1=value1,key2=value2")
	StackSetDeployCmd.Flags().StringVarP(&configFilePath, "config", "c", "", "YAML or JSON file to set tags and parameters")
	// StackSetDeployCmd.Flags().BoolVarP(&terminationProtection, "termination-protection", "t", false, "enable termination protection on the stack")
	// StackSetDeployCmd.Flags().BoolVarP(&keep, "keep", "k", false, "keep deployed resources after a failure by disabling rollbacks")
}

func readConfiguration(configFilePath string) configFormat {

	var configData configFormat

	// Read configuration file
	if len(configFilePath) != 0 {
		configFileContent, err := ioutil.ReadFile(configFilePath)
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

func mergeCliFlagsToConfigData(configData *configFormat, cliParams map[string]string, cliTags map[string]string) {
	// Merge Tags
	for k, v := range cliTags {
		if _, ok := configData.Tags[k]; ok {
			fmt.Println(console.Yellow(fmt.Sprintf("tags flag overrides tag in config file: %s", k)))
		}
		configData.Tags[k] = v
	}

	// Merge Params
	for k, v := range cliParams {
		if _, ok := configData.Parameters[k]; ok {
			fmt.Println(console.Yellow(fmt.Sprintf("params flag overrides parameter in config file: %s", k)))
		}
		configData.Parameters[k] = v
	}
}
