package builder

import "github.com/aws-cloudformation/rain/cfn/spec"

// IamBuilder contains specific code for building IAM policies
type IamBuilder struct {
	Builder
}

// NewIamBuilder creates a new IamBuilder
func NewIamBuilder() IamBuilder {
	var b IamBuilder
	b.Spec = spec.Iam

	return b
}

// Policy generates a an IAM policy body
func (b IamBuilder) Policy() (interface{}, interface{}) {
	return b.newPropertyType("", "Policy")
}
