package cmd

import (
	"testing"

	"github.com/aws-cloudformation/rain/cfn"
	"github.com/google/go-cmp/cmp"
)

func TestMergeTemplatesSuccess(t *testing.T) {
	dst := cfn.Template(map[string]interface{}{
		"AWSTemplateFormatVersion": "overwritten",
		"Description":              "Line 1",
		"Metadata": map[string]interface{}{
			"Foo": "bar",
		},
		"Transform": "AWS::Serverless",
	})

	src := cfn.Template(map[string]interface{}{
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

	expected := cfn.Template(map[string]interface{}{
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

	mergeTemplates(dst, src)

	if d := cmp.Diff(dst, expected); d != "" {
		t.Errorf(d)
	}
}

func TestMergeTemplatesClash(t *testing.T) {
	dst := cfn.Template(map[string]interface{}{
		"Description": "Line 1",
		"Metadata": map[string]interface{}{
			"Foo": "bar",
		},
	})

	src := cfn.Template(map[string]interface{}{
		"Description": "Line 2",
		"Metadata": map[string]interface{}{
			"Foo": "baz",
		},
	})

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Merge did not panic")
		}
	}()

	mergeTemplates(dst, src)
}
