package visitor

import (
	"reflect"
	"testing"

	"github.com/aws-cloudformation/rain/internal/node"
	"gopkg.in/yaml.v3"
)

// Helper function to create a simple YAML node tree for testing
func createTestYamlTree() *yaml.Node {
	// Create a structure like:
	// root (mapping)
	//  |- key1: value1
	//  |- key2:
	//       |- subkey1: subvalue1
	//       |- subkey2: subvalue2
	//  |- key3: [item1, item2, item3]

	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
	}

	// key1: value1
	key1 := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "key1"}
	value1 := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "value1"}

	// key2: {subkey1: subvalue1, subkey2: subvalue2}
	key2 := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "key2"}
	value2 := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	subkey1 := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "subkey1"}
	subvalue1 := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "subvalue1"}

	subkey2 := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "subkey2"}
	subvalue2 := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "subvalue2"}

	value2.Content = []*yaml.Node{subkey1, subvalue1, subkey2, subvalue2}

	// key3: [item1, item2, item3]
	key3 := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "key3"}
	value3 := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}

	item1 := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "item1"}
	item2 := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "item2"}
	item3 := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "item3"}

	value3.Content = []*yaml.Node{item1, item2, item3}

	// Assemble the tree
	root.Content = []*yaml.Node{key1, value1, key2, value2, key3, value3}

	return root
}

func TestNewVisitor(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}
	v := NewVisitor(node)

	if v.rootNode != node {
		t.Errorf("NewVisitor did not set rootNode correctly, got %v, want %v", v.rootNode, node)
	}

	if v.stop != false {
		t.Errorf("NewVisitor did not initialize stop correctly, got %v, want %v", v.stop, false)
	}

	if v.skip != false {
		t.Errorf("NewVisitor did not initialize skip correctly, got %v, want %v", v.skip, false)
	}

	if v.parentNode != nil {
		t.Errorf("NewVisitor did not initialize parentNode correctly, got %v, want %v", v.parentNode, nil)
	}
}

func TestGetYamlNode(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}
	v := NewVisitor(node)

	if v.GetYamlNode() != node {
		t.Errorf("GetYamlNode did not return the correct node, got %v, want %v", v.GetYamlNode(), node)
	}
}

func TestGetParentNode(t *testing.T) {
	parent := &yaml.Node{Kind: yaml.ScalarNode, Value: "parent"}
	child := &yaml.Node{Kind: yaml.ScalarNode, Value: "child"}

	v := NewVisitor(child)
	v.parentNode = parent

	if v.GetParentNode() != parent {
		t.Errorf("GetParentNode did not return the correct node, got %v, want %v", v.GetParentNode(), parent)
	}
}

func TestSkipChildren(t *testing.T) {
	v := NewVisitor(&yaml.Node{})
	v.SkipChildren()

	if !v.skip {
		t.Errorf("SkipChildren did not set skip to true, got %v, want %v", v.skip, true)
	}
}

func TestStop(t *testing.T) {
	v := NewVisitor(&yaml.Node{})
	v.Stop()

	if !v.stop {
		t.Errorf("Stop did not set stop to true, got %v, want %v", v.stop, true)
	}
}

func TestVisit(t *testing.T) {
	root := createTestYamlTree()
	v := NewVisitor(root)

	// Count the number of nodes visited
	nodeCount := 0
	v.Visit(func(node *Visitor) {
		nodeCount++
	})

	// The test tree has 14 nodes in total
	expectedNodeCount := 14
	if nodeCount != expectedNodeCount {
		t.Errorf("Visit did not visit the correct number of nodes, got %v, want %v", nodeCount, expectedNodeCount)
	}
}

func TestVisitWithStop(t *testing.T) {
	root := createTestYamlTree()
	v := NewVisitor(root)

	// Count the number of nodes visited, but stop after the first one
	nodeCount := 0
	v.Visit(func(node *Visitor) {
		nodeCount++
		node.Stop()
	})

	expectedNodeCount := 1
	if nodeCount != expectedNodeCount {
		t.Errorf("Visit with Stop did not stop correctly, visited %v nodes, want %v", nodeCount, expectedNodeCount)
	}
}

func TestVisitWithSkipChildren(t *testing.T) {
	root := createTestYamlTree()
	v := NewVisitor(root)

	// Count the number of nodes visited, but skip children of the root node
	nodeCount := 0
	v.Visit(func(node *Visitor) {
		nodeCount++
		if node.GetYamlNode() == root {
			node.SkipChildren()
		}
	})

	// Should only visit the root node
	expectedNodeCount := 1
	if nodeCount != expectedNodeCount {
		t.Errorf("Visit with SkipChildren did not skip correctly, visited %v nodes, want %v", nodeCount, expectedNodeCount)
	}
}

func TestMatch(t *testing.T) {
	root := createTestYamlTree()
	v := NewVisitor(root)

	// Find all scalar nodes with value "key1"
	results := v.Match(func(node *Visitor) bool {
		n := node.GetYamlNode()
		return n.Kind == yaml.ScalarNode && n.Value == "key1"
	})

	if len(results) != 1 {
		t.Errorf("Match did not find the correct number of nodes, got %v, want %v", len(results), 1)
	}

	if results[0].Value != "key1" {
		t.Errorf("Match did not find the correct node, got %v, want %v", results[0].Value, "key1")
	}
}

func TestMatchMultipleResults(t *testing.T) {
	root := createTestYamlTree()
	v := NewVisitor(root)

	// Find all scalar nodes
	results := v.Match(func(node *Visitor) bool {
		return node.GetYamlNode().Kind == yaml.ScalarNode
	})

	// Our test tree has 11 scalar nodes
	expectedCount := 11
	if len(results) != expectedCount {
		t.Errorf("Match did not find the correct number of scalar nodes, got %v, want %v", len(results), expectedCount)
	}
}

func TestMatchWithParentCheck(t *testing.T) {
	root := createTestYamlTree()
	v := NewVisitor(root)

	// Find all values that are direct children of the key2 mapping
	var key2Node *yaml.Node

	for i, n := range root.Content {
		if n.Value == "key2" {
			key2Node = root.Content[i+1]
		}
	}

	if key2Node == nil {
		t.Fatal("Could not find key2 node for test setup")
	}

	// Now find all nodes whose parent is the key2 value node
	results := v.Match(func(node *Visitor) bool {
		return node.GetParentNode() == key2Node
	})

	// key2's value is a mapping with 4 content nodes (2 keys and 2 values)
	expectedCount := 4
	if len(results) != expectedCount {
		t.Errorf("Match with parent check did not find the correct number of nodes, got %v, want %v, key2Node: %s", len(results), expectedCount, node.YamlStr(key2Node))
	}
}

func TestVisitOrder(t *testing.T) {
	// Create a simple tree
	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "a"},
			{Kind: yaml.ScalarNode, Value: "1"},
			{Kind: yaml.ScalarNode, Value: "b"},
			{Kind: yaml.ScalarNode, Value: "2"},
		},
	}

	v := NewVisitor(root)

	// Collect the values in the order they are visited
	var values []string
	v.Visit(func(node *Visitor) {
		n := node.GetYamlNode()
		if n.Kind == yaml.ScalarNode {
			values = append(values, n.Value)
		}
	})

	// The expected order is depth-first
	expected := []string{"a", "1", "b", "2"}
	if !reflect.DeepEqual(values, expected) {
		t.Errorf("Visit did not traverse in the expected order, got %v, want %v", values, expected)
	}
}

func TestMatchWithStop(t *testing.T) {
	root := createTestYamlTree()
	v := NewVisitor(root)

	// This should still find all matching nodes even though we stop the visitor
	// because Match uses its own visitor
	results := v.Match(func(node *Visitor) bool {
		n := node.GetYamlNode()
		if n.Kind == yaml.ScalarNode && n.Value == "key1" {
			node.Stop() // This should not affect the Match function's traversal
			return true
		}
		return false
	})

	if len(results) != 1 {
		t.Errorf("Match with Stop did not find the correct number of nodes, got %v, want %v", len(results), 1)
	}
}

func TestComplexVisitorPattern(t *testing.T) {
	// Test a more complex visitor pattern where we modify nodes during traversal

	// Create a simple tree
	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "key1"},
			{Kind: yaml.ScalarNode, Value: "value1"},
			{Kind: yaml.ScalarNode, Value: "key2"},
			{Kind: yaml.ScalarNode, Value: "value2"},
		},
	}

	v := NewVisitor(root)

	// Modify all values by appending "-modified"
	v.Visit(func(node *Visitor) {
		n := node.GetYamlNode()
		parent := node.GetParentNode()

		// If this is a value node (odd index in the parent's content)
		if parent != nil && parent.Kind == yaml.MappingNode {
			for i, child := range parent.Content {
				if i%2 == 1 && child == n && n.Kind == yaml.ScalarNode {
					n.Value = n.Value + "-modified"
				}
			}
		}
	})

	// Check that the values were modified
	modified1 := root.Content[1].Value
	modified2 := root.Content[3].Value

	if modified1 != "value1-modified" {
		t.Errorf("Complex visitor pattern did not modify first value correctly, got %v, want %v", modified1, "value1-modified")
	}

	if modified2 != "value2-modified" {
		t.Errorf("Complex visitor pattern did not modify second value correctly, got %v, want %v", modified2, "value2-modified")
	}
}
