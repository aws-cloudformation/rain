//+build !func_test

// Package aws contains functionality that wraps the AWS SDK
package aws

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/middleware"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	smithymiddleware "github.com/awslabs/smithy-go/middleware"
)

// MFAProvider is called by the AWS SDK when an MFA token number
// is required during authentication
func MFAProvider() (string, error) {
	spinner.Pause()
	defer func() {
		fmt.Println()
		spinner.Resume()
	}()

	return console.Ask("MFA Token:"), nil
}

var awsCfg *aws.Config
var creds aws.Credentials

// For debug resolver
type uaResolver string

var defaultSessionName = fmt.Sprintf("%s-%s", config.NAME, config.VERSION)
var lastSessionName = defaultSessionName

func (u uaResolver) ResolveEndpoint(service string, region string) (aws.Endpoint, error) {
	return aws.Endpoint{
		URL: string(u),
	}, nil
}

func loadConfig(ctx context.Context, sessionName string) *aws.Config {
	// Credential configs
	var configs = make([]awsconfig.Config, 0)

	// Uncomment for testing against a local endpoint
	//configs = append(configs, awsconfig.WithEndpointResolver(uaResolver("http://localhost:8000")))

	// Add user-agent
	configs = append(configs, awsconfig.WithAPIOptions(
		append(
			[]func(*smithymiddleware.Stack) error{},
			middleware.AddUserAgentKeyValue(config.NAME, config.VERSION),
		),
	))

	// Add MFA provider and Rain session name
	configs = append(configs, awsconfig.WithAssumeRoleCredentialOptions(func(options *stscreds.AssumeRoleOptions) {
		options.RoleSessionName = sessionName
		options.TokenProvider = MFAProvider
	}))

	// Supplied profile
	if config.Profile != "" {
		configs = append(configs, awsconfig.WithSharedConfigProfile(config.Profile))
	} else if p := os.Getenv("AWS_PROFILE"); p != "" {
		config.Profile = p
	}

	// Supplied region
	if config.Region != "" {
		configs = append(configs, awsconfig.WithRegion(config.Region))
	} else if r := os.Getenv("AWS_DEFAULT_REGION"); r != "" {
		config.Region = r
	}

	cfg, err := awsconfig.LoadDefaultConfig(configs...)
	if err != nil {
		panic(errors.New("unable to find valid credentials"))
	}

	// Check for validity
	creds, err = cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		config.Debugf("Error retreiving creds: %s", err.Error())
		panic(errors.New("could not establish AWS credentials; please run 'aws configure' or choose a profile"))
	}

	return &cfg
}

// Config loads an aws.Config based on current settings
func Config() aws.Config {
	return NamedConfig(defaultSessionName)
}

// NamedConfig loads an aws.Config based on current settings
// with configurable session name
func NamedConfig(sessionName string) aws.Config {
	message := "Loading AWS config"

	if creds.CanExpire && time.Until(creds.Expires) < time.Minute {
		// Check for expiry
		message = "Refreshing AWS credentials"
		awsCfg = nil
	} else if lastSessionName != sessionName {
		message = "Reloading AWS credentials"
		awsCfg = nil
	}

	if awsCfg == nil {
		spinner.Push(message)
		awsCfg = loadConfig(context.Background(), sessionName)
		spinner.Pop()
	}

	return *awsCfg
}

// SetRegion is used to set the current AWS region
func SetRegion(region string) {
	awsCfg.Region = region
}
