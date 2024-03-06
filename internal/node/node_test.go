package node_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/internal/node"
	"gopkg.in/yaml.v3"
)

func TestGetParentNotFound(t *testing.T) {
	parent := &yaml.Node{
		Kind: yaml.DocumentNode,
	}

	child := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "Child",
	}

	pair := node.GetParent(child, parent, nil)
	if pair.Value != nil {
		t.Errorf("child should not have been found")
	}
}

func TestGetParentFound(t *testing.T) {
	parent := &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: make([]*yaml.Node, 1),
	}

	childMap := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: make([]*yaml.Node, 2),
	}

	parent.Content[0] = childMap

	childKey := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "ChildKey",
	}

	childValue := &yaml.Node{
		Kind:  yaml.MappingNode,
		Value: "ChildValue",
	}

	childMap.Content[0] = childKey
	childMap.Content[1] = childValue

	pair := node.GetParent(childValue, parent, nil)
	if pair.Value != childMap {
		t.Errorf("childMap should have been found for childValue")
	}

	pair = node.GetParent(childMap, parent, nil)
	if pair.Value != parent {
		t.Errorf("parent should have been found for childMap")
	}

	childOfChild := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "ChildOfChild",
	}

	childValue.Content = make([]*yaml.Node, 4)
	childValue.Content[0] = &yaml.Node{Kind: yaml.ScalarNode, Value: "ChildOfChildKey"}
	childValue.Content[1] = childOfChild

	pair = node.GetParent(childOfChild, parent, nil)
	if pair.Value != childValue {
		t.Errorf("childValue should have been found for childOfChild")
	}
	if pair.Key != childKey {
		t.Errorf("childKey should have been found as key for childOfChild")
	}

	sequenceKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "ChildSequence"}
	childValue.Content[2] = sequenceKey
	sequence := &yaml.Node{Kind: yaml.SequenceNode, Content: make([]*yaml.Node, 2)}
	childValue.Content[3] = sequence

	sequence.Content[0] = &yaml.Node{Kind: yaml.ScalarNode, Value: "Seq0"}
	sequence.Content[1] = &yaml.Node{Kind: yaml.ScalarNode, Value: "Seq1"}

	// For a sequence, the parent Key should be  ??
	pair = node.GetParent(sequence.Content[0], parent, nil)
	if pair.Key != sequenceKey {
		t.Errorf("Seq0 pair Key should be sequenceKey")
	}
	if pair.Value != sequence {
		t.Errorf("Seq0 pair Value should be sequence")
	}

	pair = node.GetParent(sequence.Content[1], parent, nil)
	if pair.Key != sequenceKey {
		t.Errorf("Seq1 pair Key should be sequenceKey")
	}
	if pair.Value != sequence {
		t.Errorf("Seq1 pair Value should be sequence")
	}

	// Replace the sequence content with Maps
	map0 := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 2)}
	map1 := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 2)}
	sequence.Content[0] = map0
	sequence.Content[1] = map1

	map0.Content[0] = &yaml.Node{Kind: yaml.ScalarNode, Value: "Ref"}
	map0.Content[1] = &yaml.Node{Kind: yaml.ScalarNode, Value: "Foo"}
	map1.Content[0] = &yaml.Node{Kind: yaml.ScalarNode, Value: "Ref"}
	map1.Content[1] = &yaml.Node{Kind: yaml.ScalarNode, Value: "Bar"}

	pair = node.GetParent(map0.Content[1], parent, nil)
	if pair.Key != nil {
		t.Errorf("Foo pair Key should be nil")
	}

	pair = node.GetParent(map1.Content[1], parent, nil)
	if pair.Key != nil {
		t.Errorf("Bar pair Key should be nil")
	}

}

func TestRemoveFromMap(t *testing.T) {

	m := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: make([]*yaml.Node, 4),
	}

	m.Content[0] = &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "KeepKey",
	}

	m.Content[1] = &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "KeepVal",
	}

	m.Content[2] = &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "RemoveKey",
	}

	m.Content[3] = &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "RemoveVal",
	}

	err := node.RemoveFromMap(m, "RemoveKey")
	if err != nil {
		t.Error(err)
	}

	if len(m.Content) != 2 {
		t.Errorf("m.Content len is %v", len(m.Content))
	}

	if m.Content[0].Value != "KeepKey" && m.Content[1].Value != "KeepVal" {
		t.Errorf("m.Content[0] is %v, [1] is %v", m.Content[0].Value, m.Content[1].Value)
	}

}

func TestSetMapValue(t *testing.T) {
	n := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
	node.SetMapValue(n, "Test", &yaml.Node{Kind: yaml.ScalarNode, Value: "Val"})
	if len(n.Content) != 2 || n.Content[0].Value != "Test" || n.Content[1].Value != "Val" {
		t.Errorf("Unexpected length or content, len is %v", len(n.Content))
	}
	node.SetMapValue(n, "Test", &yaml.Node{Kind: yaml.ScalarNode, Value: "Val2"})
	if n.Content[1].Value != "Val2" {
		t.Errorf("Unexpected value: %v", n.Content[1].Value)
	}
}
