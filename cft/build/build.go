// Package build contains functionality to generate a cft.Template
// from specification data in cft.spec
package build

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/spec"
)

const (
	policyDocument           = "PolicyDocument"
	assumeRolePolicyDocument = "AssumeRolePolicyDocument"
	optionalTag              = "Optional"
	changeMeTag              = "CHANGEME"
)

// builder generates a template from its Spec
type builder struct {
	Spec                      spec.Spec
	IncludeOptionalProperties bool
	BuildIamPolicies          bool
}

var iam iamBuilder

func init() {
	iam = newIamBuilder()
}

func (b builder) newResource(resourceType string) (map[string]interface{}, []*cft.Comment) {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("error building resource type '%s': %w", resourceType, r))
		}
	}()

	_, ok := b.Spec[resourceType]
	if !ok {
		panic(fmt.Errorf("no such resource type '%s'", resourceType))
	}

	// Generate properties
	properties := make(map[string]interface{})
	comments := make([]*cft.Comment, 0)

	return map[string]interface{}{
		"Type":       resourceType,
		"Properties": properties,
	}, comments
}
