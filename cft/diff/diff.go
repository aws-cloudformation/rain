// Package diff provides the Diff class
// that can be used to compare CloudFormation templates
package diff

import (
	"fmt"
	"sort"
	"strings"
)

// Mode represents a diff mode
type Mode string

const (
	// Added represents a new value
	Added Mode = "+"

	// Removed represents a removed value
	Removed Mode = "-"

	// Changed represents a modified value
	Changed Mode = ">"

	// Involved represents a value that contains changes but is not wholly new itself
	Involved Mode = "|"

	// Unchanged represents a value that has not changed
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

	// String returns a string representation of the Diff
	String() string

	// Format returns a pretty-printed representation of the Diff
	// The long flag determines whether to produce long or short output
	Format(bool) string

	// Value returns the value represented by the Diff
	value() interface{}
}

// value represents a difference between values of any type
type value struct {
	val  interface{}
	mode Mode
}

// Mode returns the value's mode
func (v value) Mode() Mode {
	return v.mode
}

// value returns the value's value ;)
func (v value) value() interface{} {
	return v.val
}

// String returns a string representation of the value
func (v value) String() string {
	return fmt.Sprintf("%s%v", v.Mode(), v.value())
}

// slice represents a difference between two slices
type slice []Diff

// Mode returns the slice's mode
func (s slice) Mode() Mode {
	for _, v := range s {
		if v.Mode() != Unchanged {
			return Involved
		}
	}

	return Unchanged
}

// value returns the slice's value
func (s slice) value() interface{} {
	out := make([]interface{}, len(s))

	for i, v := range s {
		out[i] = v.value()
	}

	return out
}

// String returns a string representation of the slice
func (s slice) String() string {
	parts := make([]string, len(s))

	for i, v := range s {
		parts[i] = fmt.Sprintf("%s", v)
	}

	return fmt.Sprintf("%s[%s]", s.Mode(), strings.Join(parts, " "))
}

// dmap represents a difference between two maps
type dmap map[string]Diff

// Mode returns the dmap's mode
func (m dmap) Mode() Mode {
	s := make(slice, 0)

	for _, v := range m {
		s = append(s, v)
	}

	return s.Mode()
}

// value returns the dmap's value
func (m dmap) value() interface{} {
	out := make(map[string]interface{})

	for k, v := range m {
		out[k] = v.value()
	}

	return out
}

// keys returns the dmap's keys
func (m dmap) keys() []string {
	keys := make([]string, len(m))

	i := 0
	for k := range m {
		keys[i] = k
		i++
	}

	return keys
}

// String returns a string representation of the dmap
func (m dmap) String() string {
	keys := make([]string, 0)
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s:%s", k, m[k]))
	}

	return fmt.Sprintf("%smap[%s]", m.Mode(), strings.Join(parts, " "))
}
