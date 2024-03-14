// Code generated from Pkl module `cloudformation`. DO NOT EDIT.
package cloudformation

import (
	"context"

	"github.com/apple/pkl-go/pkl"
)

type Cloudformation struct {
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Cloudformation
func LoadFromPath(ctx context.Context, path string) (ret *Cloudformation, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Cloudformation
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (*Cloudformation, error) {
	var ret Cloudformation
	if err := evaluator.EvaluateModule(ctx, source, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}
