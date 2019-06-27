package template

import (
	"reflect"
	"sort"
	"testing"
)

func TestGraphSorting(t *testing.T) {
	graph := NewGraph()
	graph.Link("Cake", "Eggs")
	graph.Link("Cake", "Butter")
	graph.Link("Eggs", "Chicken")
	graph.Link("Dinner", "Chicken")
	graph.Link("Dinner", "Cake")

	unsorted := []interface{}{
		"Cake",
		"Eggs",
		"Butter",
		"Chicken",
		"Dinner",
	}

	// Should sort by dependency-count first, then string representation
	sorted := []interface{}{
		"Butter",
		"Chicken",
		"Eggs",
		"Cake",
		"Dinner",
	}

	// Unsorted
	if !reflect.DeepEqual(graph.Items(), unsorted) {
		t.Errorf("Graph did not retain item order: %s", graph.Items())
	}

	// Sorted
	sort.Sort(graph)
	if !reflect.DeepEqual(graph.Items(), sorted) {
		t.Errorf("Graph did not sort correctly: %s", graph.Items())
	}

	// Links should be sorted by string representation
	if !reflect.DeepEqual(graph.Get("Cake"), []interface{}{"Butter", "Eggs"}) {
		t.Errorf("Links did not sort correctly: %s", graph.Get("Instance"))
	}

	// Depth should work correctly
	if graph.Depth("Dinner") != 4 {
		t.Errorf("Depth wasn't correct: %d", graph.Depth("Dinner"))
	}
}
