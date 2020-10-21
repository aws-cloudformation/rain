package value

import (
	"fmt"
	"reflect"
)

// List represents a slice value
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

// Value returns the value of the List
func (v *List) Value() interface{} {
	out := make([]interface{}, len(v.values))
	for i, value := range v.values {
		out[i] = value.Value()
	}
	return out
}

// Get returns an element from the List
func (v *List) Get(path ...interface{}) Interface {
	if len(path) == 0 {
		return v
	}

	i, ok := path[0].(int)
	if !ok {
		panic(fmt.Errorf("lists only have int keys, not '%#v'", path[0]))
	}
	return v.values[i].Get(path[1:]...)
}

// Comment returns the List's comment
func (v *List) Comment() string {
	return v.comment
}

// SetComment sets the List's comment
func (v *List) SetComment(c string) {
	v.comment = c
}

// Nodes returns the contents of the List as a list of []Node
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
