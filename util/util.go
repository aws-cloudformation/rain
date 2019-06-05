package util

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/aws-cloudformation/rain/config"
)

func RunAttached(command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
}

func RunCapture(command string, args ...string) (string, error) {
	var out bytes.Buffer
	var err bytes.Buffer

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = &out
	cmd.Stderr = &err

	cmd.Run()

	if err.String() == "" {
		return out.String(), nil
	}

	return out.String(), fmt.Errorf(err.String())
}

func RunAwsCapture(args ...string) (string, error) {
	if config.Profile != "" {
		args = append(args, "--profile", config.Profile)
	}

	if config.Region != "" {
		args = append(args, "--region", config.Region)
	}

	return RunCapture("aws", args...)
}
