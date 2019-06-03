package format

import (
	"fmt"
)

type value struct {
	Data     interface{}
	Comments map[interface{}]interface{}
}

func (v value) Get(path []interface{}) interface{} {
	value, err := get(v.Data, path)

	if err != nil {
		panic(err)
	}

	return value
}

func (v value) GetComment(path []interface{}) string {
	value, err := get(v.Comments, path)
	comment, ok := value.(string)

	if err != nil || !ok {
		// Try looking for a root comment
		value, err = get(v.Comments, append(path, ""))
		comment, ok = value.(string)

		if err != nil || !ok {
			// Ok, there's no comment
			return ""
		}
	}

	return comment
}

func get(data interface{}, path []interface{}) (interface{}, error) {
	value := data

	for _, part := range path {
		switch v := value.(type) {
		case map[interface{}]interface{}:
			value = v[part]
		case map[string]interface{}:
			stringPart, ok := part.(string)
			if !ok {
				return nil, fmt.Errorf("Path: Invalid map key '%s'", part)
			}
			value = v[stringPart]
		case []interface{}:
			intPart, ok := part.(int)
			if !ok {
				return nil, fmt.Errorf("Path: Invalid index '%s'", part)
			}
			value = v[intPart]
		default:
			return nil, fmt.Errorf("Path: No such entry '%s'", part)
		}
	}

	return value, nil
}
