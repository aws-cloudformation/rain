package value

import (
	"fmt"
	"reflect"
	"sort"
)

type Map struct {
	values  map[string]Interface
	comment string
}

func newMap(in reflect.Value) Interface {
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

func (v *Map) Nodes() []Node {
	nodes := []Node{
		{
			Path:    []interface{}{},
			Content: v,
		},
	}

	// We'd better sort the keys!
	keys := v.Keys()
	sort.Strings(keys)

	for _, key := range keys {
		for _, child := range v.Get(key).Nodes() {
			child.Path = append([]interface{}{key}, child.Path...)
			nodes = append(nodes, child)
		}
	}

	return nodes
}
