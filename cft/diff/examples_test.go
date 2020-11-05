package diff

import (
	"fmt"
)

func ExampleValue() {
	fmt.Println(compareValues("foo", "foo"))
	fmt.Println(compareValues("foo", "bar"))

	// Output:
	// (=)foo
	// (>)bar
}

func ExampleSlice() {
	original := []interface{}{"foo"}

	fmt.Println(compareValues(original, []interface{}{"foo"}))
	fmt.Println(compareValues(original, []interface{}{"bar"}))
	fmt.Println(compareValues(original, []interface{}{}))
	fmt.Println(compareValues(original, []interface{}{"foo", "bar"}))

	// Output:
	// (=)[(=)foo]
	// (|)[(>)bar]
	// (|)[(-)foo]
	// (|)[(=)foo (+)bar]
}

func ExampleMap() {
	original := map[string]interface{}{"foo": "bar"}

	fmt.Println(compareValues(original, map[string]interface{}{"foo": "bar"}))
	fmt.Println(compareValues(original, map[string]interface{}{"foo": "baz"}))
	fmt.Println(compareValues(original, map[string]interface{}{}))
	fmt.Println(compareValues(original, map[string]interface{}{"foo": "bar", "baz": "quux"}))

	// Output:
	// (=)map[foo:(=)bar]
	// (|)map[foo:(>)baz]
	// (|)map[foo:(-)bar]
	// (|)map[baz:(+)quux foo:(=)bar]
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

	fmt.Println(compareValues(original, map[string]interface{}{
		"cake": "lie",
	}))

	// Output:
	// (|)map[cake:(+)lie foo:(-)[bar baz] quux:(-)map[mooz:xyzzy]]
}
