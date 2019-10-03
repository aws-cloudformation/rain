package value

import (
	"fmt"
	"reflect"
	"strings"
)

type Node struct {
	Path    []interface{}
	Content Interface
}

func (n Node) String() string {
	out := strings.Builder{}
	out.WriteString("[")
	for i, part := range n.Path {
		out.WriteString(fmt.Sprint(part))
		if i < len(n.Path)-1 {
			out.WriteString("/")
		}
	}
	out.WriteString("]: ")

	switch c := n.Content.(type) {
	case *Map:
		out.WriteString("{...}")
	case *List:
		out.WriteString("[...]")
	case *Scalar:
		out.WriteString(fmt.Sprint(c.Value()))
	}

	if n.Content.Comment() != "" {
		out.WriteString("  # ")
		out.WriteString(n.Content.Comment())
	}

	return out.String()
}

type Interface interface {
	Get(...interface{}) Interface
	Value() interface{}
	Comment() string
	SetComment(string)
	Nodes() []Node
}

func New(in interface{}) Interface {
	v := reflect.ValueOf(in)

	switch v.Kind() {
	case reflect.Map:
		return newMap(v)
	case reflect.Slice, reflect.Array:
		return newList(v)
	default:
		return newScalar(in)
	}
}
