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

	childValue.Content = make([]*yaml.Node, 2)
	childValue.Content[0] = &yaml.Node{Kind: yaml.ScalarNode, Value: "ChildOfChildKey"}
	childValue.Content[1] = childOfChild

	pair = node.GetParent(childOfChild, parent, nil)
	if pair.Value != childValue {
		t.Errorf("childValue should have been found for childOfChild")
	}
	if pair.Key != childKey {
		t.Errorf("childKey should have been found as key for childOfChild")
	}
}
