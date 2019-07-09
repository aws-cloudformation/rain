package value_test

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cfn/value"
)

func Example() {
	data := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": "baz",
		},
	}

	comments := map[interface{}]interface{}{
		"": "Top-level comment",
		"foo": map[string]interface{}{
			"":    "Comment on foo",
			"bar": "Comment on bar",
		},
	}

	value := value.New(data, comments)

	fmt.Println(value.Get(), "//", value.GetComment())
	fmt.Println(value.Get("foo"), "//", value.GetComment("foo"))
	fmt.Println(value.Get("foo", "bar"), "//", value.GetComment("foo", "bar"))

	// Output:
	// map[foo:map[bar:baz]] // Top-level comment
	// map[bar:baz] // Comment on foo
	// baz // Comment on bar
}
