package format

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
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

func TestGetCommonPrefix(t *testing.T) {
	testCases := []struct {
		name           string
		logicalIds     []string
		expectedPrefix string
	}{
		{
			name:           "Empty slice",
			logicalIds:     []string{},
			expectedPrefix: "",
		},
		{
			name:           "Single element",
			logicalIds:     []string{"Busket"},
			expectedPrefix: "",
		},
		{
			name:           "Common prefix",
			logicalIds:     []string{"BucketA", "BucketB", "BucketC"},
			expectedPrefix: "Bucket",
		},
		{
			name:           "No common prefix",
			logicalIds:     []string{"BucketA", "QueueB", "TableC"},
			expectedPrefix: "",
		},
		{
			name:           "Mixed case",
			logicalIds:     []string{"BucketA", "bucketB", "BucketC"},
			expectedPrefix: "",
		},
		{
			name: "TryConvertCdk",
			logicalIds: []string{
				"TryConvertCdkQueueA6B3948A",
				"TryConvertCdkQueueA6B3948B",
				"TryConvertCdkQueueA6B3948C",
				"TryConvertCdkQueuePolicy8C365983",
				"TryConvertCdkQueueTryConvertCdkStackTryConvertCdkTopic4C9C531F7A91899A",
				"TryConvertCdkTopic2CABFDF4",
				"TryConvertCdkACustomResource",
				"TryConvertCdkAnotherCustomResource",
				"TryConvertCdkAnotherCustomResource2",
			},
			expectedPrefix: "TryConvertCdk",
		},
	}

	config.Debug = true

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prefix := getCommonPrefix(tc.logicalIds)
			if prefix != tc.expectedPrefix {
				t.Errorf("Expected prefix '%s', got '%s'", tc.expectedPrefix, prefix)
			}
		})
	}
}
