package spec

import (
	"fmt"
	"reflect"
	"strings"
)

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
	for _, key := range v.MapKeys() {

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

	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).IsZero() {
			out.WriteString(fmt.Sprintf("%s: ", t.Field(i).Name))
			out.WriteString(formatValue(v.Field(i)))
			out.WriteString(",\n")
		}
	}

	out.WriteString("}")

	return out.String()
}
