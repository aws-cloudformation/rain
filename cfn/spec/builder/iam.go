package builder

import "github.com/aws-cloudformation/rain/cfn/spec"

type IamBuilder struct {
	Builder
}

func NewIamBuilder() IamBuilder {
	var b IamBuilder
	b.Spec = spec.Iam

	return b
}

func (b IamBuilder) Policy() (interface{}, interface{}) {
	return b.newPropertyType("", "Policy")
}
