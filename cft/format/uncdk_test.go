package format

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
)

func TestRemoveEmptySections(t *testing.T) {
	src := `
Parameters: {}
Resources:
  Bucket:
    Type: AWS::S3::Bucket
`
	template, err := parse.String(src)
	if err != nil {
		t.Fatal(err)
	}
	template.RemoveEmptySections()

	params, err := template.GetSection(cft.Parameters)
	if err == nil && params != nil {
		t.Fatal("expected Parameters section to be removed")
	}

}

func TestUnCDK(t *testing.T) {

	path := "../../test/templates/uncdk.yaml"
	template, err := parse.File(path)
	if err != nil {
		t.Fatalf("could not parse %s: %v", path, err)
	}

	config.Debug = true
	config.Debugf("%s", node.ToSJson(template.Node))

	expectPath := "../../test/templates/uncdk-expect.yaml"
	expectedTemplate, err := parse.File(expectPath)
	if err != nil {
		t.Fatalf("could not parse %s: %v", expectPath, err)
	}

	err = UnCDK(template)
	if err != nil {
		t.Fatal(err)
	}

	d := diff.New(template, expectedTemplate)
	if d.Mode() != "=" {
		t.Errorf("Output does not match expected: %v", d.Format(true))
	}

}
