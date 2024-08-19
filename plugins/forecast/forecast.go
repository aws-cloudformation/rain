package forecast

import (
	"fmt"
	"slices"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/aws-cloudformation/rain/plugins/deployconfig"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"gopkg.in/yaml.v3"
)

type ForecastPlugin interface {
	// GetForecasters must be implemented by plugins to return a
	// map of the prediction functions. The key is the resource type name.
	GetForecasters() map[string]func(input PredictionInput) Forecast
}

// PredictionInput is the input to forecast prediction functions
type PredictionInput struct {
	Source      cft.Template
	StackName   string
	Resource    *yaml.Node
	LogicalId   string
	StackExists bool
	Stack       types.Stack
	TypeName    string
	Dc          *deployconfig.DeployConfig
	Env         Env
	RoleArn     string
	Ignore      []string
}

// GetPropertyNode returns the node for the given property name
func (input *PredictionInput) GetPropertyNode(name string) *yaml.Node {
	_, props, _ := s11n.GetMapValue(input.Resource, "Properties")
	if props != nil {
		_, n, _ := s11n.GetMapValue(props, name)
		return n
	}
	return nil
}

// Forecast represents predictions for resources in the template
type Forecast struct {
	TypeName  string
	LogicalId string
	Passed    []Check
	Failed    []Check
	Ignore    []string
	// TODO: Errors []error
	// Instead of config.Debugf, output unexpected errors
	// Otherwise users won't know if checks are failing to run,
	// unless they view debug output
}

func MakeForecast(input *PredictionInput) Forecast {
	return Forecast{
		TypeName:  input.TypeName,
		LogicalId: input.LogicalId,
		Ignore:    input.Ignore,
		Passed:    make([]Check, 0),
		Failed:    make([]Check, 0),
	}
}

// Check is a specific check with a code that can be suppressed
type Check struct {
	Pass    bool
	Code    string
	Message string
}

func (f *Forecast) GetNumChecked() int {
	return len(f.Passed) + len(f.Failed)
}

func (f *Forecast) GetNumFailed() int {
	return len(f.Failed)
}

func (f *Forecast) GetNumPassed() int {
	return len(f.Passed)
}

func (f *Forecast) Append(forecast Forecast) {
	f.Failed = append(f.Failed, forecast.Failed...)
	f.Passed = append(f.Passed, forecast.Passed...)
}

// Add adds a pass or fail message, formatting it to include the type name and logical id
func (f *Forecast) Add(code string, passed bool, message string, lineNumber int) {
	msg := fmt.Sprintf("%v: %v %v - %v", lineNumber, f.TypeName, f.LogicalId, message)
	check := Check{
		Pass:    passed,
		Code:    code,
		Message: msg,
	}

	// If we are ignoring this check, don't add it to the forecast
	if slices.Contains(f.Ignore, code) || slices.Contains(f.Ignore, f.TypeName) {
		return
	}

	if passed {
		f.Passed = append(f.Passed, check)
	} else {
		f.Failed = append(f.Failed, check)
	}
}

func (c *Check) String() string {
	var passFail string
	if c.Pass {
		passFail = "PASS"
	} else {
		passFail = "FAIL"
	}
	return fmt.Sprintf("%s %s on line %s", c.Code, passFail, c.Message)
}

type Env struct {
	Partition string
	Region    string
	Account   string
}

// Which checks to ignore (--ignore)
var Ignore []string
