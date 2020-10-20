package client

import (
	"context"
	"fmt"
	"os"
	"time"

	rainConfig "github.com/aws-cloudformation/rain/config"
	"github.com/aws-cloudformation/rain/console"
	"github.com/aws-cloudformation/rain/console/spinner"
	"github.com/aws-cloudformation/rain/version"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/config"
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

// For debug resolver
type uaResolver string

func (u uaResolver) ResolveEndpoint(service string, region string) (aws.Endpoint, error) {
	return aws.Endpoint{
		URL: string(u),
	}, nil
}

func loadConfig(ctx context.Context) aws.Config {
	// Credential configs
	var configs = make([]config.Config, 0)

	// Uncomment for testing against a local endpoint
	//configs = append(configs, config.WithEndpointResolver(uaResolver("http://localhost:8000")))

	// Add user-agent
	configs = append(configs, config.WithAPIOptions(
		append(
			[]func(*smithymiddleware.Stack) error{},
			middleware.AddUserAgentKeyValue(version.NAME, version.VERSION),
		),
	))

	// Add MFA provider
	configs = append(configs, config.WithAssumeRoleCredentialOptions(func(options *stscreds.AssumeRoleOptions) {
		options.TokenProvider = MFAProvider
	}))

	// Supplied profile
	if rainConfig.Profile != "" {
		configs = append(configs, config.WithSharedConfigProfile(rainConfig.Profile))
	} else if p := os.Getenv("AWS_PROFILE"); p != "" {
		rainConfig.Profile = p
	}

	// Supplied region
	if rainConfig.Region != "" {
		configs = append(configs, config.WithRegion(rainConfig.Region))
	} else if r := os.Getenv("AWS_DEFAULT_REGION"); r != "" {
		rainConfig.Region = r
	}

	cfg, err := config.LoadDefaultConfig(configs...)
	if err != nil {
		panic("Unable to find valid credentials")
	}

	return cfg
}

// Config loads an aws.Config based on current settings
func Config() aws.Config {
	if awsCfg == nil {
		spinner.Status("Loading AWS config")

		cfg := loadConfig(context.Background())

		awsCfg = &cfg

		spinner.Stop()
	}

	// Check for validity
	creds, err := awsCfg.Credentials.Retrieve(context.Background())
	if err != nil {
		rainConfig.Debugf("Invalid credentials: %s", err)
		panic(err)
	}

	// Check for expiry
	if creds.CanExpire && time.Until(creds.Expires) < time.Minute {
		rainConfig.Debugf("Creds expire in less than a minute; refreshing")
		awsCfg = nil
		return Config()
	}

	return *awsCfg
}

// SetRegion is used to set the current AWS region
func SetRegion(region string) {
	awsCfg.Region = region
}
