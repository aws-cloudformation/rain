//go:generate bash generate/generate.sh
package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aws-cloudformation/rain/config"
	"github.com/aws-cloudformation/rain/util"
	"github.com/aws-cloudformation/rain/version"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/external"
)

var awsCfg *aws.Config

var defaultPython = []string{"/usr/bin/env", "python"}

func findAws() string {
	path := os.Getenv("PATH")

	for _, dir := range strings.Split(path, ":") {
		bins, err := filepath.Glob(dir + "/aws")
		if err != nil {
			panic(err)
		}

		if len(bins) == 1 {
			return bins[0]
		}
	}

	return ""
}

func getAwsPython() []string {
	aws := findAws()

	if aws == "" {
		return defaultPython
	}

	script, err := ioutil.ReadFile(aws)
	if err != nil {
		util.Debug("Couldn't load aws script: %s", err)
		return defaultPython
	}

	parts := strings.Split(string(script), "\n")

	if strings.HasPrefix(parts[0], "#!") {
		return strings.Split(parts[0][2:], " ")
	}

	return defaultPython
}

func checkConfig(cfg aws.Config) bool {
	_, err := cfg.Credentials.Retrieve()
	if err != nil {
		return false
	}

	return true
}

func loadConfig() aws.Config {
	var cfg aws.Config
	var err error

	configs := make([]external.Config, 0)

	if config.Profile != "" {
		configs = append(configs, external.WithSharedConfigProfile(config.Profile))
	}

	// Try to load just from the config
	cfg, err = external.LoadDefaultAWSConfig(configs...)
	if err != nil {
		util.Debug("Couldn't load default config: %s", err)
	} else if checkConfig(cfg) {
		util.Debug("Loaded credentials from default config")
		return cfg
	}

	// OK, let's try to load from dump_creds...
	args := getAwsPython()
	util.Debug("AWS python: %s", args)
	first := args[0]
	args = args[1:]
	args = append(args, "-c", credDumperPython)
	output, err := util.RunCapture(first, args...)
	util.Debug("Cred dumper output: %s", output)
	if err != nil {
		util.Debug("Couldn't run cred dumper: %s", err)
		panic(fmt.Errorf("Unable to load AWS config"))
	}

	// This feels more horrible the further we get down...
	var vars map[string]interface{}
	err = json.Unmarshal([]byte(output), &vars)
	if err != nil {
		util.Debug("Couldn't parse cred dumper output: %s", err)
		panic(fmt.Errorf("Unable to load AWS config"))
	}

	// But feel free to come up with a better mechanism...
	for key, value := range vars {
		// For dealing with how the aws cli loads plugins
		os.Setenv(key, fmt.Sprint(value))
	}

	// Load from the new environment variables
	cfg, err = external.LoadDefaultAWSConfig()
	if err != nil {
		util.Debug("Couldn't use cred dumper output: %s", err)
		panic(fmt.Errorf("Unable to load AWS config"))
	}

	if !checkConfig(cfg) {
		util.Debug("Unusable creds from cred dumper: %s", err)
		panic(fmt.Errorf("Unable to load AWS config"))
	}

	util.Debug("Loaded credentials from cred dumper")
	return cfg
}

func Config() aws.Config {
	if awsCfg == nil {
		util.SpinStatus("Loading AWS config...")

		cfg := loadConfig()

		// Set the user agent
		cfg.Handlers.Build.Remove(defaults.SDKVersionUserAgentHandler)
		cfg.Handlers.Build.PushFront(aws.MakeAddToUserAgentHandler(
			version.NAME,
			version.VERSION,
			runtime.Version(),
			fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			fmt.Sprintf("%s/%s", aws.SDKName, aws.SDKVersion),
		))

		if config.Region != "" {
			cfg.Region = config.Region
		}

		// For debugging
		// cfg.EndpointResolver = aws.ResolveWithEndpointURL("http://localhost:8000")

		awsCfg = &cfg

		util.SpinStop()
	}

	return *awsCfg
}

func SetRegion(region string) {
	awsCfg.Region = region
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
