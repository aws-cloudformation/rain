//go:build ignore

package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type sortableKeys []reflect.Value

func (sk sortableKeys) Len() int {
	return len(sk)
}

func (sk sortableKeys) Less(i, j int) bool {
	return sk[i].String() < sk[j].String()
}

func (sk sortableKeys) Swap(i, j int) {
	sk[i], sk[j] = sk[j], sk[i]
}

func formatValue(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Map:
		return formatMap(v.Interface())
	case reflect.Interface:
		if v.IsZero() {
			return "nil"
		}
		return formatValue(v.Elem())
	default:
		return fmt.Sprintf("%#v", v.Interface())
	}
}

func formatMap(in interface{}) string {
	t := reflect.TypeOf(in)

	out := strings.Builder{}
	out.WriteString(fmt.Sprintf("map[%s]%s",
		t.Key().String(),
		t.Elem().String(),
	))
	out.WriteString("{\n")

	v := reflect.ValueOf(in)

	// Sort keys
	keys := sortableKeys(v.MapKeys())
	sort.Sort(keys)

	for _, key := range keys {
		out.WriteString(fmt.Sprintf("%#v: ", key.Interface()))
		out.WriteString(formatValue(v.MapIndex(key)))
		out.WriteString(",\n")
	}

	out.WriteString("}")

	return out.String()
}
