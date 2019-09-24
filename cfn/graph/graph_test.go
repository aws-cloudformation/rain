package graph_test

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cfn/graph"
)

func Example_nodes() {
	g := graph.New()
	g.Add("Cake", "Eggs", "Butter")
	g.Add("Eggs", "Chicken")
	g.Add("Dinner", "Chicken", "Cake")

	fmt.Println(g.Nodes())
	// Output:
	// [Butter Chicken Eggs Cake Dinner]
}

func Example_get() {
	g := graph.New()
	g.Add("foo", "bar", "baz")
	g.Add("bar", "quux")
	g.Add("baz", "foo") // Circular dependencies are fine

	fmt.Println(g.Get("foo"))
	fmt.Println(g.Get("bar"))
	fmt.Println(g.Get("baz"))
	// Output:
	// [bar baz]
	// [quux]
	// [foo]
}

func Example_getReverse() {
	g := graph.New()
	g.Add("foo", "bar", "baz")
	g.Add("bar", "quux", "baz")
	g.Add("baz", "foo") // Circular dependencies are fine

	fmt.Println(g.GetReverse("foo"))
	fmt.Println(g.GetReverse("bar"))
	fmt.Println(g.GetReverse("baz"))
	// Output:
	// [baz]
	// [foo]
	// [bar foo]
}
