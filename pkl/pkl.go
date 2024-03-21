package pkl

import (
	"context"

	"github.com/apple/pkl-go/pkl"
	"github.com/aws-cloudformation/rain/internal/config"
)

var EvaluatorOptionsFunc = func(opts *pkl.EvaluatorOptions) {
	pkl.WithDefaultAllowedResources(opts)
	pkl.WithOsEnv(opts)
	pkl.WithDefaultAllowedModules(opts)
	pkl.WithDefaultCacheDir(opts)
	opts.Logger = pkl.NoopLogger
	opts.OutputFormat = "yaml"
}

func Yaml(filename string) (string, error) {
	// Convert the template to YAML
	evaluator, err := pkl.NewEvaluator(context.Background(), EvaluatorOptionsFunc)
	if err != nil {
		return "", err
	}
	defer evaluator.Close()
	yaml, err :=
		evaluator.EvaluateOutputText(context.Background(), pkl.FileSource(filename))
	if err != nil {
		return "", err
	}

	config.Debugf("pkl yaml: %s", yaml)

	return yaml, nil
}
