package cfn_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
)

func TestStatusIsSettled(t *testing.T) {
	for input, expected := range map[string]bool{
		"STACK_COMPLETE":     true,
		"STACK_FAILED":       true,
		"SOMETHING_COMPLETE": true,
		"SOMETHING_FAILED":   true,
		"COMPLETE_STACK":     false,
		"FAILED_STACK":       false,
	} {
		if cfn.StatusIsSettled(input) != expected {
			t.Fail()
		}
	}
}
