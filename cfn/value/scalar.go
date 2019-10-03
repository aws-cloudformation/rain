package value

import (
	"fmt"
)

type Scalar struct {
	value   interface{}
	comment string
}

func newScalar(in interface{}) Interface {
	return &Scalar{in, ""}
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

func (v *Scalar) Nodes() []Node {
	return []Node{
		{
			Path:    make([]interface{}, 0),
			Content: v,
		},
	}
}
