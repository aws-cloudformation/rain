package client

import (
	"context"
	"fmt"
	"os"
	"time"

	rainConfig "github.com/aws-cloudformation/rain/config"
	"github.com/aws-cloudformation/rain/console"
	"github.com/aws-cloudformation/rain/console/spinner"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
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

func checkCreds(cfg aws.Config, ctx context.Context) bool {
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		rainConfig.Debugf("Invalid credentials: %s", err)
		return false
	}

	if creds.CanExpire && time.Until(creds.Expires) < time.Minute {
		rainConfig.Debugf("Creds expire in less than a minute")
		return false
	}

	return true
}

func loadConfig(ctx context.Context) aws.Config {
	// Credential configs
	var configs = make([]config.Config, 0)

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

		// Set the user agent - FIXME - bring this back
		/*
			cfg.Handlers.Build.Remove(defaults.SDKVersionUserAgentHandler)
			cfg.Handlers.Build.PushFront(aws.MakeAddToUserAgentHandler(
				version.NAME,
				version.VERSION,
				runtime.Version(),
				fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
				fmt.Sprintf("%s/%s", aws.SDKName, aws.SDKVersion),
			))
		*/

		// For debugging
		// cfg.EndpointResolver = aws.ResolveWithEndpointURL("http://localhost:8000")

		awsCfg = &cfg

		spinner.Stop()
	}

	// Check for expiry
	if !checkCreds(*awsCfg, context.Background()) {
		rainConfig.Debugf("Creds are not ok; trying again")
		awsCfg = nil
		return Config()
	}

	return *awsCfg
}

// SetRegion is used to set the current AWS region
func SetRegion(region string) {
	awsCfg.Region = region
}
