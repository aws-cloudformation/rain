// Code generated from Pkl module `cloudformation`. DO NOT EDIT.
package cloudformation

import "github.com/apple/pkl-go/pkl"

type Resource interface {
	GetType() string

	GetProperties() *any

	GetCreationPolicy() *map[any]any

	GetDeletionPolicy() *string

	GetDependsOn() *[]string

	GetMetadata() *pkl.Object

	GetUpdatePolicy() *map[any]any

	GetUpdateReplacePolicy() *string

	GetCondition() *string
}

var _ Resource = (*ResourceImpl)(nil)

// A CloudFormation resource.
//
// Note that in subclasses of Resource, properties are elevated
// to the top level, so we have to rename any properties that
// conflict with resource attribute names such as `Type` and `DependsOn`.
//
// Any property that conflicts will be suffixed with `Property`.
type ResourceImpl struct {
	Type string `pkl:"Type"`

	Properties *any `pkl:"Properties"`

	CreationPolicy *map[any]any `pkl:"CreationPolicy"`

	DeletionPolicy *string `pkl:"DeletionPolicy"`

	DependsOn *[]string `pkl:"DependsOn"`

	Metadata *pkl.Object `pkl:"Metadata"`

	UpdatePolicy *map[any]any `pkl:"UpdatePolicy"`

	UpdateReplacePolicy *string `pkl:"UpdateReplacePolicy"`

	Condition *string `pkl:"Condition"`
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

func (rcv *ResourceImpl) GetMetadata() *pkl.Object {
	return rcv.Metadata
}

func (rcv *ResourceImpl) GetUpdatePolicy() *map[any]any {
	return rcv.UpdatePolicy
}

func (rcv *ResourceImpl) GetUpdateReplacePolicy() *string {
	return rcv.UpdateReplacePolicy
}

func (rcv *ResourceImpl) GetCondition() *string {
	return rcv.Condition
}
