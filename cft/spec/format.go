package spec

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
	case reflect.Ptr:
		return "&" + formatValue(v.Elem())
	case reflect.Struct:
		return formatStruct(v.Interface())
	case reflect.Map:
		return formatMap(v.Interface())
	default:
		return fmt.Sprintf("%#v", v.Interface())
	}
}

func formatMap(in interface{}) string {
	t := reflect.TypeOf(in)

	name := t.Elem().Name()
	if t.Elem().Kind() == reflect.Ptr {
		name = "*" + t.Elem().Elem().Name()
	}

	out := strings.Builder{}
	out.WriteString(fmt.Sprintf("map[%s]%s",
		t.Key().Name(),
		name,
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

func formatStruct(in interface{}) string {
	out := strings.Builder{}

	v := reflect.ValueOf(in)
	t := v.Type()

	out.WriteString(t.Name())
	out.WriteString("{\n")

	// Sort keys
	keys := make([]string, 0)
	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).IsZero() {
			keys = append(keys, t.Field(i).Name)
		}
	}

	sort.Strings(keys)

	for _, key := range keys {
		out.WriteString(fmt.Sprintf("%s: ", key))
		out.WriteString(formatValue(v.FieldByName(key)))
		out.WriteString(",\n")
	}

	out.WriteString("}")

	return out.String()
}
