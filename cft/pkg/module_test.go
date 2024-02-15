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
	runTest("test", t)
}

func TestSimple(t *testing.T) {
	runTest("simple", t)
}

func TestModInMod(t *testing.T) {
	runTest("modinmod", t)
}

// TODO: This was broken in the refactor, come back to it later
//func TestForeach(t *testing.T) {
//	runTest("foreach", t)
//}

func runTest(test string, t *testing.T) {

	// There should be 3 files for each test, for example:
	// bucket-module.yaml, bucket-template.yaml, bucket-expect.yaml

	path := fmt.Sprintf("./tmpl/%v-expect.yaml", test)

	expectedTemplate, err := parse.File(path)
	if err != nil {
		t.Errorf("expected %s: %v", test, err)
		return
	}

	pkg.Experimental = true

	packaged, err := pkg.File(fmt.Sprintf("./tmpl/%v-template.yaml", test))
	if err != nil {
		t.Errorf("packaged %s: %v", test, err)
		return
	}

	//y := format.String(packaged, format.Options{
	//	JSON:     false,
	//	Unsorted: false,
	//})

	d := diff.New(packaged, expectedTemplate)
	if d.Mode() != "=" {
		t.Errorf("Output does not match expected: %v", d.Format(true))
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
