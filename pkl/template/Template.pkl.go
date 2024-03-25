// Code generated from Pkl module `template`. DO NOT EDIT.
package template

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"github.com/aws-cloudformation/rain/pkl/cloudformation"
)

type Template struct {
	Description *string `pkl:"Description"`

	AWSTemplateFormatVersion string `pkl:"AWSTemplateFormatVersion"`

	Metadata *map[any]any `pkl:"Metadata"`

	Parameters *map[string]cloudformation.Parameter `pkl:"Parameters"`

	Resources map[string]cloudformation.Resource `pkl:"Resources"`

	Rules *map[string]cloudformation.Rule `pkl:"Rules"`

	Mappings *map[any]any `pkl:"Mappings"`

	Conditions *map[string]map[any]any `pkl:"Conditions"`

	Outputs *map[string]cloudformation.Output `pkl:"Outputs"`

	Transform *any `pkl:"Transform"`
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Template
func LoadFromPath(ctx context.Context, path string) (ret *Template, err error) {
	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		return nil, err
	}
	defer func() {
		cerr := evaluator.Close()
		if err == nil {
			err = cerr
		}
	}()
	ret, err = Load(ctx, evaluator, pkl.FileSource(path))
	return ret, err
}

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Template
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (*Template, error) {
	var ret Template
	if err := evaluator.EvaluateModule(ctx, source, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}
