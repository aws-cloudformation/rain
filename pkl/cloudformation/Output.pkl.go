// Code generated from Pkl module `cloudformation`. DO NOT EDIT.
package cloudformation

type Output interface {
	GetDescription() *any

	GetValue() any

	GetExport() *Export
}

var _ Output = (*OutputImpl)(nil)

// A stack output value
type OutputImpl struct {
	Description *any `pkl:"Description"`

	Value any `pkl:"Value"`

	Export *Export `pkl:"Export"`
}

func (rcv *OutputImpl) GetDescription() *any {
	return rcv.Description
}

func (rcv *OutputImpl) GetValue() any {
	return rcv.Value
}

func (rcv *OutputImpl) GetExport() *Export {
	return rcv.Export
}
