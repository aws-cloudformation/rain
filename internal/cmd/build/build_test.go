package build

import (
	"os"
	"testing"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
)

func fromFile(path string, rt string, short string, t *testing.T) cft.Template {
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := cfn.ParseSchema(string(source))
	if err != nil {
		t.Fatal(err)
	}
	template := startTemplate()
	resourceMap, err := template.AddMapSection(cft.Resources)
	if err != nil {
		t.Fatal(err)
	}

	r := node.AddMap(resourceMap, short)
	node.Add(r, "Type", rt)
	props := node.AddMap(r, "Properties")

	err = buildNode(props, schema, schema)
	if err != nil {
		t.Fatal(err)
	}
	return template
}

func TestBucket(t *testing.T) {
	// The actual build command has to download the schema file, so we
	// can't directly unit test it locally. Grab a cached test copy of a
	// schema and test building it.
	// TODO: Add a few more complex schemas to make sure we don't miss
	// any use cases.
	fromFile("../../../test/schemas/aws-s3-bucket.json", "AWS::S3::Bucket", "MyBucket", t)

	// TODO: Validate the output? Mostly we're happy if this didn't crash
}

func TestArrays(t *testing.T) {
	config.Debug = true
	path := "../../../test/schemas/arrays.json"
	template := fromFile(path, "AWS::Test::Arrays", "MyArrays", t)
	out := format.String(template, format.Options{
		JSON: buildJSON,
	})
	config.Debugf(out)
}
