package merge

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/google/go-cmp/cmp"
)

func TestMergeTemplatesSuccess(t *testing.T) {
	dst, _ := parse.Map(map[string]interface{}{
		"AWSTemplateFormatVersion": "overwritten",
		"Description":              "Line 1",
		"Metadata": map[string]interface{}{
			"Foo": "bar",
		},
		"Transform": "AWS::Serverless",
	})

	src, _ := parse.Map(map[string]interface{}{
		"AWSTemplateFormatVersion": "ok to overwrite",
		"Description":              "Line 2",
		"Metadata": map[string]interface{}{
			"Baz": "quux",
		},
		"Parameters": map[string]interface{}{
			"Name": map[string]interface{}{
				"Type": "String",
			},
		},
		"Transform": map[string]interface{}{
			"Name": "AWS::Include",
			"Parameters": map[string]interface{}{
				"Location": "Somewhere",
			},
		},
	})

	expected, _ := parse.Map(map[string]interface{}{
		"AWSTemplateFormatVersion": "ok to overwrite",
		"Description":              "Line 1\nLine 2",
		"Metadata": map[string]interface{}{
			"Foo": "bar",
			"Baz": "quux",
		},
		"Parameters": map[string]interface{}{
			"Name": map[string]interface{}{
				"Type": "String",
			},
		},
		"Transform": []interface{}{
			"AWS::Serverless",
			map[string]interface{}{
				"Name": "AWS::Include",
				"Parameters": map[string]interface{}{
					"Location": "Somewhere",
				},
			},
		},
	})

	actual, err := mergeTemplates(dst, src)
	if err != nil {
		t.Fatal(err)
	}

	if d := cmp.Diff(actual.Map(), expected.Map()); d != "" {
		t.Errorf(d)
	}
}

func TestMergeTemplatesClash(t *testing.T) {
	dst, _ := parse.Map(map[string]interface{}{
		"Description": "Line 1",
		"Metadata": map[string]interface{}{
			"Foo": "bar",
		},
	})

	src, _ := parse.Map(map[string]interface{}{
		"Description": "Line 2",
		"Metadata": map[string]interface{}{
			"Foo": "baz",
		},
	})

	if _, err := mergeTemplates(dst, src); err == nil {
		t.Fail()
	}
}
