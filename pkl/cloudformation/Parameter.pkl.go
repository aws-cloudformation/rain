// Code generated from Pkl module `cloudformation`. DO NOT EDIT.
package cloudformation

type Parameter interface {
	GetType() string

	GetDefault() *string

	GetAllowedValues() *[]string

	GetDescription() *string
}

var _ Parameter = (*ParameterImpl)(nil)

type ParameterImpl struct {
	Type string `pkl:"Type"`

	Default *string `pkl:"Default"`

	AllowedValues *[]string `pkl:"AllowedValues"`

	Description *string `pkl:"Description"`
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
