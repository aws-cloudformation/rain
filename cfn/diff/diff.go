// Package diff provides the Diff class
// that can be used to compare CloudFormation templates
package diff

import (
	"fmt"
	"sort"
	"strings"
)

type Mode string

const (
	Added     Mode = "+"
	Removed   Mode = "-"
	Changed   Mode = "|"
	Unchanged Mode = "="
)

func (m Mode) String() string {
	return fmt.Sprintf("(%s)", string(m))
}

// Diff is the common interface for the other types in this package.
//
// A Diff represents the difference (or lack of difference) between two values
type Diff interface {
	// Mode represents the type of change in a Diff
	Mode() Mode

	// Value returns the value represented by the Diff
	Value() interface{}

	// String returns a string representation of a Diff
	String() string
}

// Value represents a difference between values of any type
type Value struct {
	value interface{}
	mode  Mode
}

func (v Value) Mode() Mode {
	return v.mode
}

func (v Value) Value() interface{} {
	return v.value
}

func (v Value) String() string {
	return fmt.Sprint(v.Value())
}

// Slice represents a difference between two slices
type Slice []Diff

func (s Slice) Mode() Mode {
	mode := Added

	for i, v := range s {
		if i == 0 {
			mode = v.Mode()
		} else {
			if mode != v.Mode() {
				mode = Changed
			}
		}
	}

	return mode
}

func (s Slice) Value() interface{} {
	out := make([]interface{}, len(s))

	for i, v := range s {
		out[i] = v.Value()
	}

	return out
}

func (s Slice) String() string {
	parts := make([]string, len(s))

	for i, v := range s {
		parts[i] = fmt.Sprintf("%s%s", v.Mode(), v)
	}

	return fmt.Sprintf("%s[%s]", s.Mode(), strings.Join(parts, " "))
}

// Maps represents a difference between two maps
type Map map[string]Diff

func (m Map) Mode() Mode {
	s := make(Slice, 0)

	for _, v := range m {
		s = append(s, v)
	}

	return s.Mode()
}

func (m Map) Value() interface{} {
	out := make(map[string]interface{})

	for k, v := range m {
		out[k] = v.Value()
	}

	return out
}

func (m Map) Keys() []string {
	keys := make([]string, len(m))

	i := 0
	for k, _ := range m {
		keys[i] = k
		i++
	}

	return keys
}

func (m Map) String() string {
	keys := make([]string, 0)
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s%s:%s", m[k].Mode(), k, m[k]))
	}

	return fmt.Sprintf("%smap[%s]", m.Mode(), strings.Join(parts, " "))
}
