// Code generated from Pkl module `cloudformation`. DO NOT EDIT.
package cloudformation

type Resource interface {
	GetType() string

	GetProperties() *any

	GetCreationPolicy() *map[any]any

	GetDeletionPolicy() *string

	GetDependsOn() *[]string

	GetMetadata() *map[any]any

	GetUpdatePolicy() *map[any]any

	GetUpdateReplacePolicy() *string
}

var _ Resource = (*ResourceImpl)(nil)

type ResourceImpl struct {
	Type string `pkl:"Type"`

	Properties *any `pkl:"Properties"`

	CreationPolicy *map[any]any `pkl:"CreationPolicy"`

	DeletionPolicy *string `pkl:"DeletionPolicy"`

	DependsOn *[]string `pkl:"DependsOn"`

	Metadata *map[any]any `pkl:"Metadata"`

	UpdatePolicy *map[any]any `pkl:"UpdatePolicy"`

	UpdateReplacePolicy *string `pkl:"UpdateReplacePolicy"`
}

func (rcv *ResourceImpl) GetType() string {
	return rcv.Type
}

func (rcv *ResourceImpl) GetProperties() *any {
	return rcv.Properties
}

func (rcv *ResourceImpl) GetCreationPolicy() *map[any]any {
	return rcv.CreationPolicy
}

func (rcv *ResourceImpl) GetDeletionPolicy() *string {
	return rcv.DeletionPolicy
}

func (rcv *ResourceImpl) GetDependsOn() *[]string {
	return rcv.DependsOn
}

func (rcv *ResourceImpl) GetMetadata() *map[any]any {
	return rcv.Metadata
}

func (rcv *ResourceImpl) GetUpdatePolicy() *map[any]any {
	return rcv.UpdatePolicy
}

func (rcv *ResourceImpl) GetUpdateReplacePolicy() *string {
	return rcv.UpdateReplacePolicy
}
