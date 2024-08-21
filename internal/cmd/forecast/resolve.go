package forecast

import (
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws/ssm"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/aws-cloudformation/rain/plugins/deployconfig"
	fc "github.com/aws-cloudformation/rain/plugins/forecast"
	"gopkg.in/yaml.v3"
)

// recurse over properties to resolve Refs
func resolveParamRefs(name string, prop *yaml.Node, dc *deployconfig.DeployConfig, parent *yaml.Node) {
	if name == "Ref" && prop.Kind == yaml.ScalarNode {

		for _, param := range dc.Params {
			if *param.ParameterKey == prop.Value {
				if parent.Kind == yaml.MappingNode {

					var val string

					// Resolve SSM types like AWS::SSM::Parameter::Value<AWS::EC2::Image::Id>
					if param.ResolvedValue != nil {
						// Will this ever not be nil? Maybe for updates?
						val = *param.ResolvedValue
					} else {
						val = *param.ParameterValue
						// We don't have the param type here...
						if strings.HasPrefix(val, "/aws/service/") {
							// Assume this is an SSM parameter
							resolved, err := ssm.GetParameter(val)
							if err != nil {
								config.Debugf("could not get SSM parameter: %v", err)
							} else {
								val = resolved
							}
						}
					}

					// Replace the parent Mapping node
					*parent = yaml.Node{Kind: yaml.ScalarNode, Value: val}
				}
				// would it be any other Kind?
			}
		}

	} else if prop.Kind == yaml.MappingNode {
		for i := 0; i < len(prop.Content); i += 2 {
			resolveParamRefs(prop.Content[i].Value, prop.Content[i+1], dc, prop)
		}
	} else if prop.Kind == yaml.SequenceNode {
		for _, p := range prop.Content {
			resolveParamRefs("", p, dc, prop)
		}
	}
}

func resolveRefs(input fc.PredictionInput) {
	_, props, _ := s11n.GetMapValue(input.Resource, "Properties")
	if props != nil {
		for i := 0; i < len(props.Content); i += 2 {
			resolveParamRefs(props.Content[i].Value, props.Content[i+1], input.Dc, props)
		}
	}
}
