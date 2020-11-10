//+build func_test

package test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/cmd/rain"
	"github.com/aws-cloudformation/rain/internal/console"
)

func wrap(t *testing.T, args []string, expectedOut, expectedErr string, expectedCode int) {
	// Capture stdout
	realOut := os.Stdout
	ro, wo, _ := os.Pipe()
	os.Stdout = wo

	// Capture stderr
	realErr := os.Stderr
	re, we, _ := os.Pipe()
	os.Stderr = we

	console.IsTTY = false

	// Run the command
	os.Args = append([]string{
		"rain",
		"--no-colour",
		"-r",
		"mock-region-1",
	}, args...)
	exitCode := cmd.Test(rain.Cmd)

	// Reset stdout
	os.Stdout = realOut
	wo.Close()

	// Reset stderr
	os.Stderr = realErr
	we.Close()

	// Compare the output
	actualOut, _ := ioutil.ReadAll(ro)
	if d := cmp.Diff(expectedOut, string(actualOut)); d != "" {
		t.Error(d)
	}

	// Compare the err
	actualErr, _ := ioutil.ReadAll(re)
	if d := cmp.Diff(expectedErr, string(actualErr)); d != "" {
		t.Error(d)
	}

	// Compare exit code
	if exitCode != expectedCode {
		t.Errorf("Unexpected error code: %d", exitCode)
	}
}

func TestFlow(t *testing.T) {
	// Deploy
	wrap(t, []string{
		"deploy",
		"-y",
		"--params",
		"BucketName=foo",
		"templates/success.template",
	}, `Deploying template 'success.template' as stack 'success' in mock-region-1.
Stack success: CREATE_COMPLETE
  Outputs:
    MockKey: Mock value # Mock output description (exported as MockExport)
Successfully deployed success
`, "", 0)

	// Cat
	wrap(t, []string{
		"cat",
		"success",
	}, `Description: This template succeeds

Parameters:
  BucketName:
    Type: String

Resources:
  Bucket1:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Ref BucketName
`, "", 0)

	// Logs
	wrap(t, []string{
		"logs",
		"success",
	}, `Sep  9 00:00:00 success/MockResourceId (Mock::Resource::Type) CREATE_IN_PROGRESS "mock status reason"
`, "", 0)

	// List current region
	wrap(t, []string{
		"ls",
	}, `CloudFormation stacks in mock-region-1:
  success: CREATE_COMPLETE
`, "", 0)

	// List all regions
	wrap(t, []string{
		"ls",
		"-a",
	}, `CloudFormation stacks in mock-region-1:
  success: CREATE_COMPLETE
`, "", 0)

	// List stack
	wrap(t, []string{
		"ls",
		"success",
	}, `Stack success: CREATE_COMPLETE
  Outputs:
    MockKey: Mock value # Mock output description (exported as MockExport)
`, "", 0)

	// List full stack
	wrap(t, []string{
		"ls",
		"-a",
		"success",
	}, `Stack success: CREATE_COMPLETE
  Parameters:
    MockKey: Mock value
  Resources:
    MockResourceId: CREATE_COMPLETE
      MockPhysicalId
  Outputs:
    MockKey: Mock value # Mock output description (exported as MockExport)
`, "", 0)

	// Watch stack
	wrap(t, []string{
		"watch",
		"success",
	}, `Stack success: CREATE_COMPLETE
`, `not watching unchanging stack
`, 1)

	// Remove stack
	wrap(t, []string{
		"rm",
		"-y",
		"success",
	}, `Successfully deleted stack 'success'
`, "", 0)

	// List all stacks again
	wrap(t, []string{
		"ls",
		"-a",
	}, "", "", 0)
}
