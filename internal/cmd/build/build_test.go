package build

import (
	"os"
	"testing"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/node"
)

const SCHEMAS = "../../../test/schemas/"

func fromFile(path string, rt string, short string, bare bool, t *testing.T) cft.Template {
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

	ancestorTypes := make([]string, 0)
	err = buildNode(props, schema, schema, ancestorTypes, bare)
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
	fromFile(SCHEMAS+"aws-s3-bucket.json", "AWS::S3::Bucket", "MyBucket", false, t)

	// TODO: Validate the output? Mostly we're happy if this didn't crash
}

func TestArrays(t *testing.T) {
	path := SCHEMAS + "arrays.json"
	fromFile(path, "AWS::Test::Arrays", "MyArrays", false, t)
}

func TestAmplifyUIBuilderComponent(t *testing.T) {
	// This one has cycles
	path := SCHEMAS + "aws-amplifyuibuilder-component.json"
	fromFile(path, "AWS::AmplifyUIBuilder::Component", "MyComponent", false, t)
}

func TestLogGroup(t *testing.T) {
	path := SCHEMAS + "aws-logs-loggroup.json"
	fromFile(path, "AWS::Logs::LogGroup", "MyLogGroup", false, t)
}
func TestBareMetricFilter(t *testing.T) {
	path := SCHEMAS + "aws-logs-metricfilter.json"
	fromFile(path, "AWS::Logs::MetricFilter", "MyMetricFilter", true, t)
}
