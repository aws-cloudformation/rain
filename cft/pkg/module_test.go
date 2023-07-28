package pkg_test

import (
	"fmt"
	"testing"

	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/cft/pkg"
)

func TestModule(t *testing.T) {

	// There should be 3 files for each test, for example:
	// bucket-module.yaml, bucket-template.yaml, bucket-expect.yaml
	tests := []string{"bucket", "foreach"}

	for _, test := range tests {
		path := fmt.Sprintf("./%v-expect.yaml", test)

		expectedTemplate, err := parse.File(path)
		if err != nil {
			t.Error(err)
			return
		}

		pkg.Experimental = true

		packaged, err := pkg.File(fmt.Sprintf("./%v-template.yaml", test))
		if err != nil {
			t.Error(err)
			return
		}

		d := diff.New(packaged, expectedTemplate)
		if d.Mode() != "=" {
			t.Errorf("Output does not match expected: %v", d.Format(true))
		}
	}
}
