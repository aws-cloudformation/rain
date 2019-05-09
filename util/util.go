package util

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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
