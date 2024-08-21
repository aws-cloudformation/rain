package deployconfig

import "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

// DeployConfig represents the user-supplied configuration for a deployment
// This is also used by the forecast command
type DeployConfig struct {
	Params []types.Parameter
	Tags   map[string]string
}

// GetParam gets the value of a supplied parameter
func (dc DeployConfig) GetParam(name string) (string, bool) {
	for _, p := range dc.Params {
		if *p.ParameterKey == name {
			if p.ResolvedValue != nil {
				return *p.ResolvedValue, true
			}
			return *p.ParameterValue, true
		}
	}
	return "", false
}
