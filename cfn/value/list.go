package value

import (
	"fmt"
	"reflect"
)

type List struct {
	values  []Interface
	comment string
}

func newList(in reflect.Value) Interface {
	out := List{
		values: make([]Interface, in.Len()),
	}

	for i := 0; i < in.Len(); i++ {
		out.values[i] = New(in.Index(i).Interface())
	}

	return &out
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

func (v *List) Nodes() []Node {
	nodes := []Node{
		{
			Path:    []interface{}{},
			Content: v,
		},
	}

	for i, value := range v.values {
		for _, child := range value.Nodes() {
			child.Path = append([]interface{}{i}, child.Path...)
			nodes = append(nodes, child)
		}
	}

	return nodes
}
