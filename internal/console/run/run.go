// Package run contains utility functions for executing external commands
package run

import (
	"os"
	"os/exec"
)

// Attached runs the specified command with Stdin, Stdout, and Stderr attached
// to their default locations (usually the console)
func Attached(command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
}

// Capture runs the specified command and returns the contents of the command's
// Stdout as a string. Additionally, if there is any output from the command
// in Stderr, an error will be returned.
func Capture(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)

	out, err := cmd.Output()

	return string(out), err
}
