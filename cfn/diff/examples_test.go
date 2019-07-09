package diff_test

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cfn/diff"
)

func ExampleValue() {
	fmt.Println(diff.New("foo", "foo"))
	fmt.Println(diff.New("foo", "bar"))

	// Output:
	// foo
	// bar
}

func ExampleSlice() {
	original := []interface{}{"foo"}

	fmt.Println(diff.New(original, []interface{}{"foo"}))
	fmt.Println(diff.New(original, []interface{}{"bar"}))
	fmt.Println(diff.New(original, []interface{}{}))
	fmt.Println(diff.New(original, []interface{}{"foo", "bar"}))

	// Output:
	// (=)[(=)foo]
	// (|)[(|)bar]
	// (-)[(-)foo]
	// (|)[(=)foo (+)bar]
}

func ExampleMap() {
	original := map[string]interface{}{"foo": "bar"}

	fmt.Println(diff.New(original, map[string]interface{}{"foo": "bar"}))
	fmt.Println(diff.New(original, map[string]interface{}{"foo": "baz"}))
	fmt.Println(diff.New(original, map[string]interface{}{}))
	fmt.Println(diff.New(original, map[string]interface{}{"foo": "bar", "baz": "quux"}))

	// Output:
	// (=)map[(=)foo:bar]
	// (|)map[(|)foo:baz]
	// (-)map[(-)foo:bar]
	// (|)map[(+)baz:quux (=)foo:bar]
}

func ExampleNew() {
	original := map[string]interface{}{
		"foo": []interface{}{
			"bar",
			"baz",
		},
		"quux": map[string]interface{}{
			"mooz": "xyzzy",
		},
	}

	fmt.Println(diff.New(original, map[string]interface{}{
		"cake": "lie",
	}))

	// Output:
	// (|)map[(+)cake:lie (-)foo:[bar baz] (-)quux:map[mooz:xyzzy]]
}
