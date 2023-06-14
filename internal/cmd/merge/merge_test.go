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
			"AWS::CloudFormation::Interface": map[string]interface{}{
				"ParameterGroups": []interface{}{
					map[string]interface{}{
						"Label": map[string]interface{}{
							"default": "Network Configuration",
						},
						"Parameters": []interface{}{
							"VPCID",
							"SubnetId",
							"SecurityGroupID",
						},
					},
				},
				"ParameterLabels": map[string]interface{}{
					"VPCID": map[string]interface{}{
						"default": "Which VPC should this be deployed to?",
					},
				},
			},
			"Foo": "bar",
		},
		"Transform": "AWS::Serverless",
	})

	src, _ := parse.Map(map[string]interface{}{
		"AWSTemplateFormatVersion": "ok to overwrite",
		"Description":              "Line 2",
		"Metadata": map[string]interface{}{
			"AWS::CloudFormation::Interface": map[string]interface{}{
				"ParameterGroups": []interface{}{
					map[string]interface{}{
						"Label": map[string]interface{}{
							"default": "Amazon EC2 Configuration",
						},
						"Parameters": []interface{}{
							"InstanceType",
							"KeyName",
						},
					},
				},
				"ParameterLabels": map[string]interface{}{
					"KeyName": map[string]interface{}{
						"default": "EC2 Instance Ker Pair",
					},
				},
			},
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
			"AWS::CloudFormation::Interface": map[string]interface{}{
				"ParameterGroups": []interface{}{
					map[string]interface{}{
						"Label": map[string]interface{}{
							"default": "Network Configuration",
						},
						"Parameters": []interface{}{
							"VPCID",
							"SubnetId",
							"SecurityGroupID",
						},
					},
					map[string]interface{}{
						"Label": map[string]interface{}{
							"default": "Amazon EC2 Configuration",
						},
						"Parameters": []interface{}{
							"InstanceType",
							"KeyName",
						},
					},
				},
				"ParameterLabels": map[string]interface{}{
					"VPCID": map[string]interface{}{
						"default": "Which VPC should this be deployed to?",
					},
					"KeyName": map[string]interface{}{
						"default": "EC2 Instance Ker Pair",
					},
				},
			},
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

	forceMerge = false
	actual, err := mergeTemplates(dst, src)
	if err != nil {
		t.Fatal(err)
	}

	if d := cmp.Diff(actual.Map(), expected.Map()); d != "" {
		t.Errorf(d)
	}
}

func TestForceMergeTemplatesSuccess(t *testing.T) {
	dst, _ := parse.Map(map[string]interface{}{
		"AWSTemplateFormatVersion": "overwritten",
		"Description":              "Line 1",
		"Metadata": map[string]interface{}{
			"AWS::CloudFormation::Interface": map[string]interface{}{
				"ParameterLabels": map[string]interface{}{
					"VPCID": map[string]interface{}{
						"default": "Which VPC should this be deployed to?",
					},
				},
			},
			"Foo": "bar",
		},
		"Parameters": map[string]interface{}{
			"Name": map[string]interface{}{
				"Type": "String",
			},
		},
	})

	src, _ := parse.Map(map[string]interface{}{
		"AWSTemplateFormatVersion": "ok to overwrite",
		"Description":              "Line 2",
		"Metadata": map[string]interface{}{
			"AWS::CloudFormation::Interface": map[string]interface{}{
				"ParameterLabels": map[string]interface{}{
					"VPCID": map[string]interface{}{
						"default": "Which VPC should this be deployed to?",
					},
				},
			},
			"Foo": "quux",
		},
		"Parameters": map[string]interface{}{
			"Name": map[string]interface{}{
				"Type": "String",
			},
		},
	})

	expected, _ := parse.Map(map[string]interface{}{
		"AWSTemplateFormatVersion": "ok to overwrite",
		"Description":              "Line 1\nLine 2",
		"Metadata": map[string]interface{}{
			"AWS::CloudFormation::Interface": map[string]interface{}{
				"ParameterLabels": map[string]interface{}{
					"VPCID": map[string]interface{}{
						"default": "Which VPC should this be deployed to?",
					},
					"VPCID_2": map[string]interface{}{
						"default": "Which VPC should this be deployed to?",
					},
				},
			},
			"Foo":   "bar",
			"Foo_2": "quux",
		},
		"Parameters": map[string]interface{}{
			"Name": map[string]interface{}{
				"Type": "String",
			},
			"Name_2": map[string]interface{}{
				"Type": "String",
			},
		},
	})

	forceMerge = true
	actual, err := mergeTemplates(dst, src)
	if err != nil {
		t.Fatal(err)
	}

	if d := cmp.Diff(actual.Map(), expected.Map()); d != "" {
		t.Errorf(d)
	}
}

func TestEmptyMergeTemplatesSuccess(t *testing.T) {
	src, _ := parse.Map(map[string]interface{}{
		"AWSTemplateFormatVersion": "ok to overwrite",
		"Description":              "Line 2",
		"Metadata": map[string]interface{}{
			"AWS::CloudFormation::Interface": map[string]interface{}{
				"ParameterGroups": []interface{}{
					map[string]interface{}{
						"Label": map[string]interface{}{
							"default": "Amazon EC2 Configuration",
						},
						"Parameters": []interface{}{
							"InstanceType",
							"KeyName",
						},
					},
				},
				"ParameterLabels": map[string]interface{}{
					"KeyName": map[string]interface{}{
						"default": "EC2 Instance Ker Pair",
					},
				},
			},

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
		"Resources": map[string]interface{}{
			"Type": "AWS::SSM::Parameter",
			"Properties": map[string]interface{}{
				"Name":  "test",
				"Type":  "String",
				"Value": "Value",
			},
		},
	})

	empty, _ := parse.Map(map[string]interface{}{})

	forceMerge = false
	// rain merge src.yaml /dev/null
	{
		actual, err := mergeTemplates(src, empty)
		if err != nil {
			t.Fatal(err)
		}

		if d := cmp.Diff(actual.Map(), src.Map()); d != "" {
			t.Errorf(d)
		}
	}

	// rain merge /dev/null src.yaml
	{
		actual, err := mergeTemplates(empty, src)
		if err != nil {
			t.Fatal(err)
		}

		if d := cmp.Diff(actual.Map(), src.Map()); d != "" {
			t.Errorf(d)
		}
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

	forceMerge = false
	if _, err := mergeTemplates(dst, src); err == nil {
		t.Fail()
	}
}
