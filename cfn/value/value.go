package value

import (
	"fmt"
	"reflect"
)

type Value interface {
	Get(...interface{}) Value
	Value() interface{}
	Comment() string
	SetComment(string)
}

type scalarValue struct {
	value   interface{}
	comment string
}

type mapValue struct {
	values  map[string]Value
	comment string
}

type listValue struct {
	values  []Value
	comment string
}

func New(in interface{}) Value {
	v := reflect.ValueOf(in)

	switch v.Kind() {
	case reflect.Map:
		return newMapValue(v)
	case reflect.Slice, reflect.Array:
		return newListValue(v)
	default:
		return newScalarValue(in)
	}
}

func newScalarValue(in interface{}) Value {
	return &scalarValue{in, ""}
}

func newMapValue(in reflect.Value) Value {
	if in.Type().Key().String() != "string" {
		panic(fmt.Errorf("s11n only supports maps with string keys, no: %T", in.Interface()))
	}

	out := mapValue{
		values: make(map[string]Value),
	}

	for _, key := range in.MapKeys() {
		out.values[key.String()] = New(in.MapIndex(key).Interface())
	}

	return &out
}

func newListValue(in reflect.Value) Value {
	out := listValue{
		values: make([]Value, in.Len()),
	}

	for i := 0; i < in.Len(); i++ {
		out.values[i] = New(in.Index(i).Interface())
	}

	return &out
}

func (v *scalarValue) Value() interface{} {
	return v.value
}

func (v *scalarValue) Get(path ...interface{}) Value {
	if len(path) != 0 {
		panic(fmt.Errorf("Attempt to index (%v) scalar: %v", path, v.value))
	}

	return v
}

func (v *scalarValue) Comment() string {
	return v.comment
}

func (v *scalarValue) SetComment(c string) {
	v.comment = c
}

func (v *mapValue) Value() interface{} {
	out := make(map[string]interface{}, len(v.values))
	for key, value := range v.values {
		out[key] = value.Value()
	}
	return out
}

func (v *mapValue) Get(path ...interface{}) Value {
	if len(path) == 0 {
		return v
	}

	s, ok := path[0].(string)
	if !ok {
		panic(fmt.Errorf("Maps only have string keys, not: %#v", path[0]))
	}

	out, ok := v.values[s]
	if !ok {
		return nil
	}

	return out.Get(path[1:]...)
}

func (v *mapValue) Comment() string {
	return v.comment
}

func (v *mapValue) SetComment(c string) {
	v.comment = c
}

func (v *listValue) Value() interface{} {
	out := make([]interface{}, len(v.values))
	for i, value := range v.values {
		out[i] = value.Value()
	}
	return out
}

func (v *listValue) Get(path ...interface{}) Value {
	if len(path) == 0 {
		return v
	}

	i, ok := path[0].(int)
	if !ok {
		panic(fmt.Errorf("Lists only have int keys, not: %#v", path[0]))
	}
	return v.values[i].Get(path[1:]...)
}

func (v *listValue) Comment() string {
	return v.comment
}

func (v *listValue) SetComment(c string) {
	v.comment = c
}
