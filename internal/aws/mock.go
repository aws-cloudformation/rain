//+build func_test

// Package aws contains functionality that wraps the AWS SDK
package aws

import (
	"context"
	"os"

	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws/aws-sdk-go-v2/aws"
)

var awsCfg *aws.Config

func loadConfig(ctx context.Context) aws.Config {
	cfg := aws.Config{}

	if config.Region != "" {
		cfg.Region = config.Region
	} else if r := os.Getenv("AWS_DEFAULT_REGION"); r != "" {
		cfg.Region = r
	}

	return cfg
}

// Config loads an aws.Config based on current settings
func Config() aws.Config {
	if awsCfg == nil {
		spinner.Push("Loading AWS config")

		cfg := loadConfig(context.Background())

		awsCfg = &cfg

		spinner.Pop()
	}

	return *awsCfg
}

// SetRegion is used to set the current AWS region
func SetRegion(region string) {
	awsCfg.Region = region
}
