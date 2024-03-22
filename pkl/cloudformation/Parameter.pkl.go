// Code generated from Pkl module `cloudformation`. DO NOT EDIT.
package cloudformation

type Parameter interface {
	GetType() string

	GetDefault() *string

	GetAllowedValues() *[]string

	GetDescription() *string

	GetAllowedPattern() *string

	GetConstraintDescription() *string

	GetMaxLength() *int

	GetMaxValue() *float64

	GetMinLength() *int

	GetMinValue() *float64

	GetNoEcho() *bool
}

var _ Parameter = (*ParameterImpl)(nil)

// A CloudFormation Parameter
type ParameterImpl struct {
	Type string `pkl:"Type"`

	Default *string `pkl:"Default"`

	AllowedValues *[]string `pkl:"AllowedValues"`

	Description *string `pkl:"Description"`

	AllowedPattern *string `pkl:"AllowedPattern"`

	ConstraintDescription *string `pkl:"ConstraintDescription"`

	MaxLength *int `pkl:"MaxLength"`

	MaxValue *float64 `pkl:"MaxValue"`

	MinLength *int `pkl:"MinLength"`

	MinValue *float64 `pkl:"MinValue"`

	NoEcho *bool `pkl:"NoEcho"`
}

func (rcv *ParameterImpl) GetType() string {
	return rcv.Type
}

func (rcv *ParameterImpl) GetDefault() *string {
	return rcv.Default
}

func (rcv *ParameterImpl) GetAllowedValues() *[]string {
	return rcv.AllowedValues
}

func (rcv *ParameterImpl) GetDescription() *string {
	return rcv.Description
}

func (rcv *ParameterImpl) GetAllowedPattern() *string {
	return rcv.AllowedPattern
}

func (rcv *ParameterImpl) GetConstraintDescription() *string {
	return rcv.ConstraintDescription
}

func (rcv *ParameterImpl) GetMaxLength() *int {
	return rcv.MaxLength
}

func (rcv *ParameterImpl) GetMaxValue() *float64 {
	return rcv.MaxValue
}

func (rcv *ParameterImpl) GetMinLength() *int {
	return rcv.MinLength
}

func (rcv *ParameterImpl) GetMinValue() *float64 {
	return rcv.MinValue
}

func (rcv *ParameterImpl) GetNoEcho() *bool {
	return rcv.NoEcho
}
