package node_test

import (
	"fmt"
	"testing"

	"github.com/aws-cloudformation/rain/cft/parse"
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

func TestDiff(t *testing.T) {

	original := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
	override := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}

	original.Content = append(original.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "A"})
	original.Content = append(original.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "foo"})

	override.Content = append(override.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "A"})
	override.Content = append(override.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "bar"})

	diff := node.Diff(original, override)
	if len(diff) != 2 {
		t.Fatalf("should have one difference, got %d", len(diff))
	}

	override.Content[1].Value = "foo"
	diff = node.Diff(original, override)
	if len(diff) != 0 {
		t.Fatalf("should have no difference, got %d", len(diff))
	}

}

func TestMergeNodes(t *testing.T) {
	original := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
	override := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
	expected := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}

	original.Content = append(original.Content, node.MakeScalar("A"))
	original.Content = append(original.Content, node.MakeScalar("foo"))
	original.Content = append(original.Content, node.MakeScalar("B"))
	original.Content = append(original.Content, node.MakeSequence([]string{"1", "2"}))

	override.Content = append(override.Content, node.MakeScalar("A"))
	override.Content = append(override.Content, node.MakeScalar("bar"))
	override.Content = append(override.Content, node.MakeScalar("B"))
	override.Content = append(override.Content, node.MakeSequence([]string{"3", "4"}))

	expected.Content = append(expected.Content, node.MakeScalar("A"))
	expected.Content = append(expected.Content, node.MakeScalar("bar"))
	expected.Content = append(expected.Content, node.MakeScalar("B"))
	expected.Content = append(expected.Content,
		node.MakeSequence([]string{"1", "2", "3", "4"}))

	merged := node.MergeNodes(original, override)

	diff := node.Diff(merged, expected)

	if len(diff) > 0 {
		for _, d := range diff {
			fmt.Println(d)
		}
		t.Fatalf("nodes are not the same")
	}

}

func TestMergeNodes2(t *testing.T) {
	original := `
A: aaa
B: bbb
C:
Obj:
  D: ddd
E:
- 1
- 2
F:
- Key: a
  Val: b
- Key: x
  Val: y
`

	override := `
A: AAA
B: bbb
C:
Obj:
  D: DDD
E:
- 3
- 4
F:
- Key: a
  Val: c
`

	expect := `
A: AAA
B: bbb
C:
Obj:
  D: DDD
E:
- 1
- 2
- 3
- 4
F:
- Key: a
  Val: c
- Key: x
  Val: y
`

	originalTemplate, err := parse.String(original)
	if err != nil {
		t.Fatal(err)
	}

	overrideTemplate, err := parse.String(override)
	if err != nil {
		t.Fatal(err)
	}

	expectedTemplate, err := parse.String(expect)
	if err != nil {
		t.Fatal(err)
	}

	merged := node.MergeNodes(originalTemplate.Node.Content[0], overrideTemplate.Node.Content[0])
	diff := node.Diff(merged, expectedTemplate.Node.Content[0])
	if len(diff) > 0 {
		for _, d := range diff {
			fmt.Println(d)
		}
		t.Fatalf("nodes are not the same")
	}
}

func TestAddMap(t *testing.T) {
	parent := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
	m := node.AddMap(parent, "Test")
	if m == nil {
		t.Errorf("AddMap returned nil")
	}
	mm := node.AddMap(parent, "Test")
	if m != mm {
		t.Errorf("AddMap second add should return the same object")
	}
}

func TestMergeNodesWithDifferentKinds(t *testing.T) {
	// Test merging nodes of different kinds (should return override)
	original := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
	override := &yaml.Node{Kind: yaml.SequenceNode, Content: make([]*yaml.Node, 0)}

	original.Content = append(original.Content, node.MakeScalar("A"))
	original.Content = append(original.Content, node.MakeScalar("foo"))

	override.Content = append(override.Content, node.MakeScalar("1"))
	override.Content = append(override.Content, node.MakeScalar("2"))

	merged := node.MergeNodes(original, override)

	if merged.Kind != yaml.SequenceNode {
		t.Fatalf("Expected merged node to be a sequence node, got %v", merged.Kind)
	}

	if len(merged.Content) != 2 {
		t.Fatalf("Expected merged content length to be 2, got %d", len(merged.Content))
	}

	if merged.Content[0].Value != "1" || merged.Content[1].Value != "2" {
		t.Fatalf("Merged content doesn't match override content")
	}
}

func TestMergeNodesWithScalars(t *testing.T) {
	// Test merging scalar nodes (should return override)
	original := node.MakeScalar("original")
	override := node.MakeScalar("override")

	merged := node.MergeNodes(original, override)

	if merged.Value != "override" {
		t.Fatalf("Expected merged value to be 'override', got '%s'", merged.Value)
	}
}

func TestMergeNodesWithNestedMaps(t *testing.T) {
	original := `
Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: original-bucket
      Tags:
        - Key: Environment
          Value: Dev
        - Key: Project
          Value: Test
`

	override := `
Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: override-bucket
      Tags:
        - Key: Environment
          Value: Prod
        - Key: Owner
          Value: Team1
`

	expect := `
Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: override-bucket
      Tags:
        - Key: Environment
          Value: Prod
        - Key: Project
          Value: Test
        - Key: Owner
          Value: Team1
`

	originalTemplate, err := parse.String(original)
	if err != nil {
		t.Fatal(err)
	}

	overrideTemplate, err := parse.String(override)
	if err != nil {
		t.Fatal(err)
	}

	expectedTemplate, err := parse.String(expect)
	if err != nil {
		t.Fatal(err)
	}

	merged := node.MergeNodes(originalTemplate.Node.Content[0], overrideTemplate.Node.Content[0])
	diff := node.Diff(merged, expectedTemplate.Node.Content[0])
	if len(diff) > 0 {
		for _, d := range diff {
			fmt.Println(d)
		}
		t.Fatalf("nodes are not the same")
	}
}

func TestMergeNodesWithEmptyNodes(t *testing.T) {
	// Test with empty original node
	emptyOriginal := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
	override := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}

	override.Content = append(override.Content, node.MakeScalar("A"))
	override.Content = append(override.Content, node.MakeScalar("foo"))

	merged := node.MergeNodes(emptyOriginal, override)

	if len(merged.Content) != 2 {
		t.Fatalf("Expected merged content length to be 2, got %d", len(merged.Content))
	}

	if merged.Content[0].Value != "A" || merged.Content[1].Value != "foo" {
		t.Fatalf("Merged content doesn't match override content")
	}

	// Test with empty override node
	original := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
	emptyOverride := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}

	original.Content = append(original.Content, node.MakeScalar("B"))
	original.Content = append(original.Content, node.MakeScalar("bar"))

	merged = node.MergeNodes(original, emptyOverride)

	if len(merged.Content) != 2 {
		t.Fatalf("Expected merged content length to be 2, got %d", len(merged.Content))
	}

	if merged.Content[0].Value != "B" || merged.Content[1].Value != "bar" {
		t.Fatalf("Merged content doesn't match original content")
	}
}
