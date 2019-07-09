// Package value provides a Value type that can be used to contain structured data
// of any type and comments that reference elements within it.
package value

import (
	"fmt"
)

func get(data interface{}, path []interface{}) (interface{}, error) {
	out := data

	for _, part := range path {
		switch v := out.(type) {
		case map[interface{}]interface{}:
			out = v[part]
		case map[string]interface{}:
			stringPart, ok := part.(string)
			if !ok {
				return nil, fmt.Errorf("Path: Invalid map key '%s'", part)
			}
			out = v[stringPart]
		case []interface{}:
			intPart, ok := part.(int)
			if !ok {
				return nil, fmt.Errorf("Path: Invalid index '%s'", part)
			}
			out = v[intPart]
		default:
			return nil, fmt.Errorf("Path: No such entry '%s'", part)
		}
	}

	return out, nil
}

// Value holds structured data of any type and associated comments.
// Comments should be in the same format as the data it relates to
// with the exception that a map key of the empty string ("")
// will be taken to mean a comment related to the map as a whole
type Value struct {
	data     interface{}
	comments map[interface{}]interface{}
}

// New creates a new Value from the supplied data and comments
func New(data interface{}, comments map[interface{}]interface{}) Value {
	return Value{
		data:     data,
		comments: comments,
	}
}

// Get returns part of the Value's data by using a path given as a slice.
// The slice should contain map keys and array indexes that identify where the data is.
//
// For eaxmple: '"foo", 1' would return the value from index 1 of an array
// that is stored with they key foo in a map.
func (v Value) Get(path ...interface{}) interface{} {
	out, err := get(v.data, path)

	if err != nil {
		panic(err)
	}

	return out
}

// GetComment returns the comment matching the path provided.
// If no comment is found at the exact path, GetComment will try
// looking for a comment with a map key of ""
// as a special case where the associated data is a map
func (v Value) GetComment(path ...interface{}) string {
	value, err := get(v.comments, path)
	comment, ok := value.(string)

	if err != nil || !ok {
		// Try looking for a root comment
		value, err = get(v.comments, append(path, ""))
		comment, ok = value.(string)

		if err != nil || !ok {
			// Ok, there's no comment
			return ""
		}
	}

	return comment
}
