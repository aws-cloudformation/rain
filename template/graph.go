package template

import (
	"fmt"
	"sort"
)

type Link struct {
	From interface{}
	To   interface{}
}

type links []interface{}

func (l links) Len() int {
	return len(l)
}

func (l links) Less(i, j int) bool {
	return fmt.Sprint(l[i]) < fmt.Sprint(l[j])
}

func (l links) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// A collection of elements and their dependencies
type Graph struct {
	graph map[interface{}]map[interface{}]bool
	order []interface{}
}

func NewGraph() Graph {
	return Graph{
		graph: make(map[interface{}]map[interface{}]bool, 0),
		order: make([]interface{}, 0),
	}
}

func (g *Graph) Add(item interface{}) {
	if _, ok := g.graph[item]; !ok {
		g.graph[item] = make(map[interface{}]bool)
		g.order = append(g.order, item)
	}
}

func (g *Graph) Link(from, to interface{}) {
	g.Add(from)
	g.Add(to)

	g.graph[from][to] = true
}

func (g Graph) Len() int {
	return len(g.order)
}

func (g Graph) Depth(item interface{}) int {
	seen := map[interface{}]bool{
		item: true,
	}

	count := 0

	var dive func(interface{})

	dive = func(from interface{}) {
		for to, _ := range g.graph[from] {
			if !seen[to] {
				seen[to] = true
				count++

				dive(to)
			}
		}
	}

	dive(item)

	return count
}

func (g Graph) Less(i, j int) bool {
	a, b := g.order[i], g.order[j]

	diff := g.Depth(a) - g.Depth(b)

	if diff == 0 {
		return fmt.Sprint(a) < fmt.Sprint(b)
	}

	return diff < 0
}

func (g Graph) Swap(i, j int) {
	g.order[i], g.order[j] = g.order[j], g.order[i]
}

func (g Graph) Items() []interface{} {
	return g.order
}

func (g Graph) Get(from interface{}) []interface{} {
	links := make(links, len(g.graph[from]))

	i := 0
	for to, _ := range g.graph[from] {
		links[i] = to
		i++
	}

	sort.Sort(links)

	return links
}
