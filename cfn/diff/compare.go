package diff

import (
	"math"
	"reflect"
)

// New returns a Diff that represents
// the difference between two values of any type
//
// To be able to compare slices and maps recursively, they must of type
// []interface{} and map[string]interface{}, respectively
func New(old, new interface{}) Diff {
	if reflect.TypeOf(old) != reflect.TypeOf(new) {
		return Value{new, Changed}
	}

	switch v := old.(type) {
	case []interface{}:
		return compareSlices(v, new.([]interface{}))
	case map[string]interface{}:
		return compareMaps(v, new.(map[string]interface{}))
	default:
		if !reflect.DeepEqual(old, new) {
			return Value{new, Changed}
		}
	}

	return Value{old, Unchanged}
}

func compareSlices(old, new []interface{}) Diff {
	max := int(math.Max(float64(len(old)), float64(len(new))))
	d := make(Slice, max)

	for i := 0; i < max; i++ {
		if i >= len(old) {
			d[i] = Value{new[i], Added}
		} else if i >= len(new) {
			d[i] = Value{old[i], Removed}
		} else {
			d[i] = New(old[i], new[i])
		}
	}

	return d
}

func compareMaps(old, new map[string]interface{}) Diff {
	d := make(Map)

	// New and updated keys
	for key, value := range new {
		if _, ok := old[key]; !ok {
			d[key] = Value{value, Added}
		} else {
			d[key] = New(old[key], value)
		}
	}

	// Removed keys
	for key, value := range old {
		if _, ok := new[key]; !ok {
			d[key] = Value{value, Removed}
		}
	}

	return d
}
