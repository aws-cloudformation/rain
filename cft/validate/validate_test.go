package validate

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/google/go-cmp/cmp"
)

func wrap(data map[string]interface{}) errors {
	t, err := parse.Map(data)
	if err != nil {
		errs := make(errors, 0)
		errs.add(err.Error())
		return errs
	}

	return Template(t)
}

func check(t *testing.T, expected, actual errors) {
	if d := cmp.Diff(expected, actual); d != "" {
		t.Error(d)
	}
}

func TestNoResources(t *testing.T) {
	expected := errors{
		{Path: nil, Value: "Template has no resources"},
	}

	actual := wrap(map[string]interface{}{})

	check(t, expected, actual)
}

func TestBadResources(t *testing.T) {
	expected := errors{
		{Path: []interface{}{"Resources"}, Value: "Resources must be a map"},
	}

	actual := wrap(map[string]interface{}{"Resources": false})

	check(t, expected, actual)
}

func TestBadResource(t *testing.T) {
	expected := errors{
		{Path: []interface{}{"Resources", "Bucket"}, Value: "Resource must be a map"},
	}

	actual := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"Bucket": false,
		},
	})

	check(t, expected, actual)
}

func TestNoResourceType(t *testing.T) {
	expected := errors{
		{Path: []interface{}{"Resources", "Bucket"}, Value: "Resource must have a Type"},
	}

	actual := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"Bucket": map[string]interface{}{},
		},
	})

	check(t, expected, actual)
}

func TestBadResourceType(t *testing.T) {
	expected := errors{
		{Path: []interface{}{"Resources", "Bucket", "Type"}, Value: "Type must be a string"},
	}

	actual := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"Bucket": map[string]interface{}{
				"Type": false,
			},
		},
	})

	check(t, expected, actual)
}

func TestUnknownResourceType(t *testing.T) {
	expected := errors{
		{Path: []interface{}{"Resources", "Bucket", "Type"}, Value: "Unknown type 'SWA::3S::Tekcub'"},
	}

	actual := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"Bucket": map[string]interface{}{
				"Type": "SWA::3S::Tekcub",
			},
		},
	})

	check(t, expected, actual)
}

func TestBadResourceProperties(t *testing.T) {
	expected := errors{
		{Path: []interface{}{"Resources", "Bucket", "Properties"}, Value: "Properties must be a map"},
	}

	actual := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"Bucket": map[string]interface{}{
				"Type":       "AWS::S3::Bucket",
				"Properties": false,
			},
		},
	})

	check(t, expected, actual)
}

func TestUnknownResourceProperty(t *testing.T) {
	expected := errors{
		{Path: []interface{}{"Resources", "Bucket", "Properties", "BananaPhone"}, Value: "Unknown property 'BananaPhone'"},
	}

	actual := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"Bucket": map[string]interface{}{
				"Type": "AWS::S3::Bucket",
				"Properties": map[string]interface{}{
					"BananaPhone": true,
				},
			},
		},
	})

	check(t, expected, actual)
}

func TestMultipleBadResourceTypes(t *testing.T) {
	expected := errors{
		{Path: []interface{}{"Resources", "Bucket1", "Type"}, Value: "Type must be a string"},
		{Path: []interface{}{"Resources", "Bucket2", "Type"}, Value: "Type must be a string"},
	}

	actual := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"Bucket1": map[string]interface{}{
				"Type": false,
			},
			"Bucket2": map[string]interface{}{
				"Type": false,
			},
		},
	})

	check(t, expected, actual)
}

func TestMissingProperties(t *testing.T) {
	expected := errors{
		{Path: []interface{}{"Resources", "RT", "Properties"}, Value: "Missing required properties: VpcId"},
	}

	actual := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"RT": map[string]interface{}{
				"Type": "AWS::EC2::RouteTable",
			},
		},
	})

	check(t, expected, actual)
}
