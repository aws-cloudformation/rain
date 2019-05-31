package client

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/aws-cloudformation/rain/config"
	"github.com/aws-cloudformation/rain/util"
	"github.com/aws-cloudformation/rain/version"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
)

var awsCfg *aws.Config

func GetConfig() aws.Config {
	if awsCfg == nil {
		configs := make([]external.Config, 0)

		if config.Profile != "" {
			configs = append(configs, external.WithSharedConfigProfile(config.Profile))
		}

		cfg, err := external.LoadDefaultAWSConfig(configs...)
		if err != nil {
			util.Die(fmt.Errorf("Unable to load AWS config: %s", err))
		}

		// Set the user agent
		cfg.Handlers.Build.Remove(defaults.SDKVersionUserAgentHandler)
		cfg.Handlers.Build.PushFront(aws.MakeAddToUserAgentHandler(
			version.NAME,
			version.VERSION,
			runtime.Version(),
			runtime.GOOS,
			runtime.GOARCH,
		))

		if config.Region != "" {
			cfg.Region = config.Region
		}

		if cfg.Region == "" {
			util.Die(errors.New("Unable to load AWS config"))
		}

		awsCfg = &cfg
	}

	return *awsCfg
}

type Error error

func NewError(err error) Error {
	if err == nil {
		return nil
	}

	if err, ok := err.(awserr.Error); ok {
		return Error(errors.New(err.Message()))
	}

	return Error(err)
}
