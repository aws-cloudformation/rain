package cfn_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/cfn"
	"github.com/aws-cloudformation/rain/cfn/value"
)

func wrap(data map[string]interface{}) (value.Interface, bool) {
	return cfn.Template(data).Check()
}

func TestNoResources(t *testing.T) {
	out, ok := wrap(map[string]interface{}{})

	if ok || out.Comment() != "Missing Resources!" {
		t.Fail()
	}
}

func TestBadResources(t *testing.T) {
	out, ok := wrap(map[string]interface{}{"Resources": false})

	if ok || out.Get("Resources").Comment() != "Not a map!" {
		t.Fail()
	}
}

func TestBadResource(t *testing.T) {
	out, ok := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"Bucket": false,
		},
	})

	if ok || out.Get("Resources", "Bucket").Comment() != "Not a map!" {
		t.Fail()
	}
}

func TestNoResourceType(t *testing.T) {
	out, ok := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"Bucket": map[string]interface{}{},
		},
	})

	if ok || out.Get("Resources", "Bucket").Comment() != "Missing Type!" {
		t.Fail()
	}
}

func TestBadResourceType(t *testing.T) {
	out, ok := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"Bucket": map[string]interface{}{
				"Type": false,
			},
		},
	})

	if ok || out.Get("Resources", "Bucket", "Type").Comment() != "Invalid type!" {
		t.Fail()
	}
}

func TestUnknownResourceType(t *testing.T) {
	out, ok := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"Bucket": map[string]interface{}{
				"Type": "SWA::3S::Tekcub",
			},
		},
	})

	if ok || out.Get("Resources", "Bucket", "Type").Comment() != "Unknown type" {
		t.Fail()
	}
}

func TestBadResourceProperties(t *testing.T) {
	out, ok := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"Bucket": map[string]interface{}{
				"Type":       "AWS::S3::Bucket",
				"Properties": false,
			},
		},
	})

	if ok || out.Get("Resources", "Bucket", "Properties").Comment() != "Not a map!" {
		t.Fail()
	}
}

func TestUnknownResourceProperty(t *testing.T) {
	out, ok := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"Bucket": map[string]interface{}{
				"Type": "AWS::S3::Bucket",
				"Properties": map[string]interface{}{
					"BananaPhone": true,
				},
			},
		},
	})

	if ok || out.Get("Resources", "Bucket", "Properties", "BananaPhone").Comment() != "Unknown property" {
		t.Fail()
	}
}

func TestMultipleBadResourceTypes(t *testing.T) {
	out, ok := wrap(map[string]interface{}{
		"Resources": map[string]interface{}{
			"Bucket1": map[string]interface{}{
				"Type": false,
			},
			"Bucket2": map[string]interface{}{
				"Type": false,
			},
		},
	})

	if ok || out.Get("Resources", "Bucket1", "Type").Comment() != "Invalid type!" {
		t.Fail()
	}

	if ok || out.Get("Resources", "Bucket2", "Type").Comment() != "Invalid type!" {
		t.Fail()
	}
}
