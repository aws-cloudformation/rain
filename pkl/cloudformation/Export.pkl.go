// Code generated from Pkl module `cloudformation`. DO NOT EDIT.
package cloudformation

type Export interface {
	GetName() any
}

var _ Export = (*ExportImpl)(nil)

// A stack Output exported value
type ExportImpl struct {
	Name any `pkl:"Name"`
}

func (rcv *ExportImpl) GetName() any {
	return rcv.Name
}
