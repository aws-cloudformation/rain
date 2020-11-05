//+build func_test

package test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/aws-cloudformation/rain/internal/cmd/rain"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/google/go-cmp/cmp"
)

func wrap(t *testing.T, args []string, expected string) {
	// Capture stdout
	realOut := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	console.IsTTY = false

	// Run the command
	os.Args = append([]string{
		"rain",
		"--no-colour",
		"-r",
		"mock-region-1",
	}, args...)
	rain.Cmd.Execute()

	// Compare the output
	os.Stdout = realOut
	w.Close()
	actual, _ := ioutil.ReadAll(r)

	if d := cmp.Diff(expected, string(actual)); d != "" {
		t.Error(d)
	}
}

func TestFlow(t *testing.T) {
	// Deploy
	wrap(t, []string{
		"deploy",
		"-f",
		"--params",
		"BucketName=foo",
		"templates/success.template",
	}, `Preparing template 'success.template'
Loading AWS config
Checking current status of stack 'success'
Creating change set
Deploying template 'success.template' as stack 'success' in mock-region-1.

Stack success: CREATE_COMPLETE
  Outputs:
    MockKey: Mock value # Mock output description (exported as MockExport)

Successfully deployed success
`)

	// Cat
	wrap(t, []string{
		"cat",
		"success",
	}, `Getting template from stack 'success'
Description: This template succeeds

Parameters:
  BucketName:
    Type: String

Resources:
  Bucket1:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Ref BucketName
`)

	// Logs
	wrap(t, []string{
		"logs",
		"success",
	}, `Getting logs for stack 'success'
mock logical resource id:  # Mock::Resource::Type
- CREATE_IN_PROGRESS "mock status reason"
`)

	// List current region
	wrap(t, []string{
		"ls",
	}, `Fetching stacks in mock-region-1
CloudFormation stacks in mock-region-1:
  success: CREATE_COMPLETE
`)

	// List all regions
	wrap(t, []string{
		"ls",
		"-a",
	}, `Fetching region list
Fetching stacks in mock-region-1
CloudFormation stacks in mock-region-1:
  success: CREATE_COMPLETE

Fetching stacks in mock-region-2
Fetching stacks in mock-region-3
`)

	// List stack
	wrap(t, []string{
		"ls",
		"success",
	}, `Fetching stack status
Stack success: CREATE_COMPLETE
  Outputs:
    MockKey: Mock value # Mock output description (exported as MockExport)
`)

	// List full stack
	wrap(t, []string{
		"ls",
		"-a",
		"success",
	}, `Fetching stack status
Stack success: CREATE_COMPLETE
  Parameters:
    MockKey: Mock value

  Resources:
    Mock logical resource id: CREATE_COMPLETE
      Mock physical resource id

  Outputs:
    MockKey: Mock value # Mock output description (exported as MockExport)
`)

	// Watch stack
	wrap(t, []string{
		"watch",
		"success",
	}, `Stack success: CREATE_COMPLETE
Not watching unchanging stack.
`)

	// Remove stack
	wrap(t, []string{
		"rm",
		"-f",
		"success",
	}, `Fetching stack status
Deleting stack 'success' in mock-region-1
Successfully deleted stack 'success'
`)

	// List all stacks
	wrap(t, []string{
		"ls",
		"-a",
	}, `Fetching region list
Fetching stacks in mock-region-1
Fetching stacks in mock-region-2
Fetching stacks in mock-region-3
`)
}
