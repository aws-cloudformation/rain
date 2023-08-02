package pkg_test

import (
	"fmt"
	"testing"

	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/cft/pkg"
	"gopkg.in/yaml.v3"
)

func TestModule(t *testing.T) {

	// There should be 3 files for each test, for example:
	// bucket-module.yaml, bucket-template.yaml, bucket-expect.yaml
	tests := []string{"bucket", "foreach"}

	for _, test := range tests {
		path := fmt.Sprintf("./tmpl/%v-expect.yaml", test)

		expectedTemplate, err := parse.File(path)
		if err != nil {
			t.Error(err)
			return
		}

		pkg.Experimental = true

		packaged, err := pkg.File(fmt.Sprintf("./tmpl/%v-template.yaml", test))
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

func TestCsvToSequence(t *testing.T) {
	csv := "A,B,C"
	seq := pkg.ConvertCsvToSequence(csv)
	if seq == nil || seq.Kind != yaml.SequenceNode {
		t.Errorf("expected a sequence node")
	}
	if seq.Content[0].Value != "A" ||
		seq.Content[1].Value != "B" ||
		seq.Content[2].Value != "C" {
		t.Errorf("Unexpected sequence")
	}
}
