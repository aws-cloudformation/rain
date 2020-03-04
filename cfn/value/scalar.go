package value

import (
	"fmt"
)

// Scalar represents an arbitrary (scalar) interface{} value
type Scalar struct {
	value   interface{}
	comment string
}

func newScalar(in interface{}) Interface {
	return &Scalar{in, ""}
}

// Value returns the value of the Scalar
func (v *Scalar) Value() interface{} {
	return v.value
}

// Get returns the Scalar's value. Only an empty path is valid.
func (v *Scalar) Get(path ...interface{}) Interface {
	if len(path) != 0 {
		panic(fmt.Errorf("Attempt to index (%v) scalar: %v", path, v.value))
	}

	return v
}

// Comment returns the Scalar's comment
func (v *Scalar) Comment() string {
	return v.comment
}

// SetComment sets the Scalar's comment
func (v *Scalar) SetComment(c string) {
	v.comment = c
}

// Nodes returns the Scalar's value in a list of []Node
func (v *Scalar) Nodes() []Node {
	return []Node{
		{
			Path:    make([]interface{}, 0),
			Content: v,
		},
	}
}
