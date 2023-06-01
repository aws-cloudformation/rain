// The dc package contains types and functions to facilitate parsing
// user-supplied configuration like tags and parameters, which
// are used for the deploy and forecast commands
package dc

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/ptr"
	"gopkg.in/yaml.v2"
)

type configFileFormat struct {
	Parameters map[string]string `yaml:"Parameters"`
	Tags       map[string]string `yaml:"Tags"`
}

// DeployConfig represents the user-supplied configuration for a deployment
// This is also used by the forecast command
type DeployConfig struct {
	Params []types.Parameter
	Tags   map[string]string
}

// GetParameters checks the combined params supplied as args and in a file
// and asks the user to supply any values that are missing
func GetParameters(
	template cft.Template,
	combinedParameters map[string]string,
	old []types.Parameter,
	stackExists bool,
	yes bool) []types.Parameter {

	newParams := make([]types.Parameter, 0)

	oldMap := make(map[string]types.Parameter)
	for _, param := range old {
		// Ignore NoEcho values
		if stackExists || ptr.ToString(param.ParameterValue) != "****" {
			oldMap[ptr.ToString(param.ParameterKey)] = param
		}
	}

	if params, ok := template.Map()["Parameters"]; ok {
		// Check we don't have any unknown params
		for k := range combinedParameters {
			if _, ok := params.(map[string]interface{})[k]; !ok {
				fmt.Println(console.Yellow(fmt.Sprintf("unknown parameter: %s", k)))
			}
		}

		// Decide on a default value
		for k, p := range params.(map[string]interface{}) {
			// New variable so we don't mess up the pointers below
			param := p.(map[string]interface{})

			value := ""
			usePrevious := false

			// Decide if we have an existing value
			if cliParam, ok := combinedParameters[k]; ok {
				value = cliParam
			} else {
				extra := ""

				if oldParam, ok := oldMap[k]; ok {
					extra = fmt.Sprintf(" (existing value: %s)", fmt.Sprint(*oldParam.ParameterValue))

					if stackExists {
						usePrevious = true
					} else {
						value = *oldParam.ParameterValue
					}
				} else if defaultValue, ok := param["Default"]; ok {
					extra = fmt.Sprintf(" (default value: %s)", fmt.Sprint(defaultValue))
					value = fmt.Sprint(defaultValue)
				} else if yes {
					panic(fmt.Errorf("no default or existing value for parameter '%s'. Set a default, supply a --params flag, or deploy without the --yes flag", k))
				}

				if !yes {
					spinner.Pause()

					prompt := fmt.Sprintf("Enter a value for parameter '%s'", k)

					if description, ok := param["Description"]; ok {
						prompt += fmt.Sprintf(" \"%s\"", description)
					}

					prompt += fmt.Sprintf("%s:", extra)

					newValue := console.Ask(prompt)
					if newValue != "" {
						value = newValue
						usePrevious = false
					}
				}
			}

			if usePrevious {
				newParams = append(newParams, types.Parameter{
					ParameterKey:     ptr.String(k),
					UsePreviousValue: ptr.Bool(true),
				})
			} else {
				newParams = append(newParams, types.Parameter{
					ParameterKey:   ptr.String(k),
					ParameterValue: ptr.String(value),
				})
			}
		}
	}

	spinner.Resume()

	return newParams
}

var FixStackNameRe *regexp.Regexp

const MaxStackNameLength = 128

// ListToMap converts a pflag parsed StringSlice into a map
// where values are expected to be presented in the form
// Foo=bar,Baz=quux,mooz,Xyzzy=garply
func ListToMap(name string, in []string) map[string]string {
	out := make(map[string]string, len(in))
	lastKey := ""
	for _, v := range in {
		parts := strings.SplitN(v, "=", 2)

		if len(parts) != 2 {
			if lastKey == "" {
				panic(fmt.Errorf("unable to parse %s: %s", name, v))
			} else {
				out[lastKey] += "," + parts[0]
			}
		} else {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			if _, ok := out[key]; ok {
				panic(fmt.Errorf("duplicate %s: %s", name, key))
			}

			out[key] = value

			lastKey = key
		}
	}

	return out
}

// GetStackName returns stackName if it is not blank, otherwise it creates
// a name based on the template file name
func GetStackName(stackName string, base string) string {
	if stackName == "" {
		stackName = base[:len(base)-len(filepath.Ext(base))]

		// Now ensure it's a valid cfc name
		stackName = FixStackNameRe.ReplaceAllString(stackName, "-")

		if len(stackName) > MaxStackNameLength {
			stackName = stackName[:MaxStackNameLength]
		}
	}
	return stackName
}

// GetDeployConfig populates an instance of DeployConfig based on user-supplied values
func GetDeployConfig(
	tags []string,
	params []string,
	configFilePath string,
	base string,
	template cft.Template,
	stack types.Stack,
	stackExists bool,
	yes bool) (*DeployConfig, error) {

	dc := &DeployConfig{}

	// Parse tags
	parsedTagFlag := ListToMap("tag", tags)

	// Parse params
	parsedParamFlag := ListToMap("param", params)

	var combinedTags map[string]string
	var combinedParameters map[string]string

	if len(configFilePath) != 0 {
		configFileContent, err := os.ReadFile(configFilePath)
		if err != nil {
			panic(ui.Errorf(err, "unable to read config file '%s'", configFilePath))
		}

		var configFile configFileFormat
		err = yaml.Unmarshal([]byte(configFileContent), &configFile)
		if err != nil {
			panic(ui.Errorf(err, "unable to parse yaml in '%s'", configFilePath))
		}

		combinedTags = configFile.Tags
		combinedParameters = configFile.Parameters

		for k, v := range parsedTagFlag {
			if _, ok := combinedTags[k]; ok {
				fmt.Println(console.Yellow(fmt.Sprintf("tags flag overrides tag in config file: %s", k)))
			}
			combinedTags[k] = v
		}

		for k, v := range parsedParamFlag {
			if _, ok := combinedParameters[k]; ok {
				fmt.Println(console.Yellow(fmt.Sprintf("params flag overrides parameter in config file: %s", k)))
			}
			combinedParameters[k] = v
		}
	} else {
		combinedTags = parsedTagFlag
		combinedParameters = parsedParamFlag
	}

	dc.Tags = combinedTags

	// Parse params
	config.Debugf("Handling parameters")
	parameters := GetParameters(template, combinedParameters,
		stack.Parameters, stackExists, yes)

	if config.Debug {
		for _, param := range parameters {
			val := ptr.ToString(param.ParameterValue)
			if ptr.ToBool(param.UsePreviousValue) {
				val = "<previous value>"
			}
			config.Debugf("  %s: %s", ptr.ToString(param.ParameterKey), val)
		}
	}

	dc.Params = parameters

	return dc, nil
}

// converts map of strings to a slice of types.Tag
func MakeTags(tags map[string]string) []types.Tag {
	out := make([]types.Tag, 0)

	for key, value := range tags {
		out = append(out, types.Tag{
			Key:   ptr.String(key),
			Value: ptr.String(value),
		})
	}

	return out
}

func init() {
	FixStackNameRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)
}
