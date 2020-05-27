package models

import (
	"fmt"
	"reflect"
	"strings"
)

func formatValue(v reflect.Value) string {
	switch v.Kind() {
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

	out := strings.Builder{}
	out.WriteString(fmt.Sprintf("map[%s]models.%s",
		t.Key().Name(),
		t.Elem().Name(),
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

	out.WriteString("models.")
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
