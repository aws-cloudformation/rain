package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/aws-cloudformation/rain/config"
	"github.com/aws-cloudformation/rain/console"
	"github.com/aws-cloudformation/rain/console/spinner"
	"github.com/aws-cloudformation/rain/version"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
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
		config.Debugf("Invalid credentials: %s", err)
		return false
	}

	if creds.CanExpire && time.Until(creds.Expires) < time.Minute {
		config.Debugf("Creds expire in less than a minute")
		return false
	}

	return true
}

func tryConfig(ctx context.Context, configs external.Configs, resolvers []external.AWSConfigResolver) (aws.Config, bool) {
	cfg, err := configs.ResolveAWSConfig(resolvers)
	if err != nil {
		config.Debugf("Credentials failed: %s", err)
		return cfg, false
	}

	return cfg, checkCreds(cfg, ctx)
}

func loadConfig(ctx context.Context) aws.Config {
	var cfg aws.Config
	var err error
	var ok bool

	// Default resolver set as used in the SDK
	var resolvers = []external.AWSConfigResolver{
		external.ResolveDefaultAWSConfig,
		external.ResolveHandlersFunc,
		external.ResolveEndpointResolverFunc,
		external.ResolveCustomCABundle,
		external.ResolveEnableEndpointDiscovery,
		external.ResolveRegion,
		external.ResolveEC2Region,
		external.ResolveDefaultRegion,
		external.ResolveCredentials,
	}

	// Minimal configs
	var configs external.Configs = []external.Config{
		external.WithMFATokenFunc(MFAProvider),
	}

	if config.Profile != "" {
		configs = append(configs, external.WithSharedConfigProfile(config.Profile))
	} else if os.Getenv("AWS_PROFILE") != "" {
		config.Profile = os.Getenv("AWS_PROFILE")
	}

	if config.Region != "" {
		configs = append(configs, external.WithRegion(config.Region))
	}

	configs, err = configs.AppendFromLoaders(external.DefaultConfigLoaders)
	if err != nil {
		panic(err)
	}

	config.Debugf("Trying default configs...")
	if cfg, ok = tryConfig(ctx, configs, resolvers); ok {
		config.Debugf("...and they're valid")
		return cfg
	}

	panic("Unable to find valid credentials")
}

// Config loads an aws.Config based on current settings
func Config() aws.Config {
	if awsCfg == nil {
		spinner.Status("Loading AWS config")

		cfg := loadConfig(context.Background())

		// Set the user agent
		cfg.Handlers.Build.Remove(defaults.SDKVersionUserAgentHandler)
		cfg.Handlers.Build.PushFront(aws.MakeAddToUserAgentHandler(
			version.NAME,
			version.VERSION,
			runtime.Version(),
			fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			fmt.Sprintf("%s/%s", aws.SDKName, aws.SDKVersion),
		))

		// For debugging
		// cfg.EndpointResolver = aws.ResolveWithEndpointURL("http://localhost:8000")

		awsCfg = &cfg

		spinner.Stop()
	}

	// Check for expiry
	if !checkCreds(*awsCfg, context.Background()) {
		config.Debugf("Creds are not ok; trying again")
		awsCfg = nil
		return Config()
	}

	return *awsCfg
}

// SetRegion is used to set the current AWS region
func SetRegion(region string) {
	awsCfg.Region = region
}

// Error is used to wrap errors thrown from the client package
type Error error

// NewError wraps a standard error value as a client.Error
func NewError(err error) Error {
	if err == nil {
		return nil
	}

	if err, ok := err.(awserr.Error); ok {
		return Error(errors.New(err.Message()))
	}

	return Error(err)
}
