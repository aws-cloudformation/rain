package value

import (
	"fmt"
	"reflect"
)

type Interface interface {
	Get(...interface{}) Interface
	Value() interface{}
	Comment() string
	SetComment(string)
}

type Scalar struct {
	value   interface{}
	comment string
}

type Map struct {
	values  map[string]Interface
	comment string
}

type List struct {
	values  []Interface
	comment string
}

func New(in interface{}) Interface {
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

func newScalarValue(in interface{}) Interface {
	return &Scalar{in, ""}
}

func newMapValue(in reflect.Value) Interface {
	if in.Type().Key().String() != "string" {
		panic(fmt.Errorf("s11n only supports maps with string keys, no: %T", in.Interface()))
	}

	out := Map{
		values: make(map[string]Interface),
	}

	for _, key := range in.MapKeys() {
		out.values[key.String()] = New(in.MapIndex(key).Interface())
	}

	return &out
}

func newListValue(in reflect.Value) Interface {
	out := List{
		values: make([]Interface, in.Len()),
	}

	for i := 0; i < in.Len(); i++ {
		out.values[i] = New(in.Index(i).Interface())
	}

	return &out
}

func (v *Scalar) Value() interface{} {
	return v.value
}

func (v *Scalar) Get(path ...interface{}) Interface {
	if len(path) != 0 {
		panic(fmt.Errorf("Attempt to index (%v) scalar: %v", path, v.value))
	}

	return v
}

func (v *Scalar) Comment() string {
	return v.comment
}

func (v *Scalar) SetComment(c string) {
	v.comment = c
}

func (v *Map) Value() interface{} {
	out := make(map[string]interface{}, len(v.values))
	for key, value := range v.values {
		out[key] = value.Value()
	}
	return out
}

func (v *Map) Get(path ...interface{}) Interface {
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

func (v *Map) Comment() string {
	return v.comment
}

func (v *Map) SetComment(c string) {
	v.comment = c
}

func (v *Map) Keys() []string {
	out := make([]string, 0)
	for key, _ := range v.values {
		out = append(out, key)
	}
	return out
}

func (v *List) Value() interface{} {
	out := make([]interface{}, len(v.values))
	for i, value := range v.values {
		out[i] = value.Value()
	}
	return out
}

func (v *List) Get(path ...interface{}) Interface {
	if len(path) == 0 {
		return v
	}

	i, ok := path[0].(int)
	if !ok {
		panic(fmt.Errorf("Lists only have int keys, not: %#v", path[0]))
	}
	return v.values[i].Get(path[1:]...)
}

func (v *List) Comment() string {
	return v.comment
}

func (v *List) SetComment(c string) {
	v.comment = c
}
