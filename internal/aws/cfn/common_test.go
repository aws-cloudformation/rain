package cfn_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
)

func TestUniqueStrings(t *testing.T) {
	in := []string{"a", "a", "b", "c"}
	expect := []string{"a", "b", "c"}
	uq := cfn.UniqueStrings(in)
	if uq[0] != expect[0] || uq[1] != expect[1] || uq[2] != expect[2] {
		t.Error()
	}
}
