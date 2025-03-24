package node

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/aws-cloudformation/rain/internal/config"
	"gopkg.in/yaml.v3"
)

// NodePair represents a !!map key-value pair
type NodePair struct {
	Key   *yaml.Node
	Value *yaml.Node

	// Parent is used by modules to reference the parent template resource
	Parent *NodePair
}

func (np *NodePair) String() string {
	return fmt.Sprintf("Key: %v\nValue: %v",
		ToSJson(np.Key), ToSJson(np.Value))
}

// Clone returns a copy of the provided node
func Clone(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	out := &yaml.Node{
		Kind:        node.Kind,
		Style:       node.Style,
		Tag:         node.Tag,
		Value:       node.Value,
		Anchor:      node.Anchor,
		Alias:       Clone(node.Alias),
		Content:     make([]*yaml.Node, len(node.Content)),
		HeadComment: node.HeadComment,
		LineComment: node.LineComment,
		FootComment: node.FootComment,
		Line:        node.Line,
		Column:      node.Column,
	}

	for i, child := range node.Content {
		out.Content[i] = Clone(child)
	}

	return out
}

// Returns the parent node of node, paired with its name if it's a map pair.
// If it is not a map pair, only NodePair.Value is not nil.
// (YAML maps are arrays with even indexes being names and odd indexes being values)
//
// node
//
//	Content
//	  0: Name
//	  1: Map
//	    Content
//	      0: a
//	      1: b
//
// In the above, if I want b's parent node pair, I get [Name, Map]
// This allows us to ask "what is the parent's name",
// which is useful for knowing the logical name of the resource a node belongs to.
//
// For sequence elements that are maps, the Key will be nil
func GetParent(node *yaml.Node, rootNode *yaml.Node, priorNode *yaml.Node) NodePair {
	if node == rootNode {
		config.Debugf("getParent node and rootNode are the same")
		return NodePair{Key: node, Value: node}
	}

	if node == nil {
		config.Debugf("node is nil")
		return NodePair{nil, nil, nil}
	}

	var found *yaml.Node
	var before *yaml.Node
	var pair NodePair

	if rootNode.Kind == yaml.DocumentNode || rootNode.Kind == yaml.SequenceNode {
		for _, n := range rootNode.Content {
			if n == node {
				found = rootNode
				before = priorNode
				break
			}
			pair = GetParent(node, n, nil)
			if pair.Value != nil {
				found = pair.Value
				before = pair.Key
				break
			}
		}
	} else if rootNode.Kind == yaml.MappingNode {
		for i := 0; i < len(rootNode.Content); i += 2 {
			n := rootNode.Content[i+1]
			if n == node {
				found = rootNode
				before = priorNode
				break
			}
			pair = GetParent(node, n, rootNode.Content[i])
			if pair.Value != nil {
				found = pair.Value
				before = pair.Key
				break
			}
		}
	}
	return NodePair{Key: before, Value: found}
}

type SNode struct {
	Kind    string
	Value   string
	Anchor  string   `json:",omitempty"`
	Content []*SNode `json:",omitempty"`
}

func makeSNode(node *yaml.Node) *SNode {
	var k string
	switch node.Kind {
	case yaml.DocumentNode:
		k = "Document"
	case yaml.SequenceNode:
		k = "Sequence"
	case yaml.MappingNode:
		k = "Mapping"
	case yaml.ScalarNode:
		k = "Scalar"
	case yaml.AliasNode:
		k = "Alias"
	default:
		k = "?"
	}

	content := make([]*SNode, 0)
	if node.Content != nil {
		for _, child := range node.Content {
			if child == nil {
				content = append(content, &SNode{Kind: "?", Value: "nil!"})
			} else {
				content = append(content, makeSNode(child))
			}
		}
	}

	s := SNode{Kind: k, Value: node.Value, Content: content, Anchor: node.Anchor}
	return &s
}

// Convert a node to a shortened JSON for easier debugging
func ToSJson(node *yaml.Node) string {
	if node == nil {
		return "nil"
	}
	j, err := json.MarshalIndent(makeSNode(node), "", "  ")
	if err != nil {
		return fmt.Sprintf("Failed to marshal node to short json: %v:", err)
	}
	return string(j)
}

// Convert a node to JSON
func ToJson(node *yaml.Node) string {
	j, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		return fmt.Sprintf("Failed to marshal node to json: %v:", err)
	}
	return string(j)
}

// Convert a node to a YAML string for troubleshooting
func YamlStr(node *yaml.Node) string {
	if node == nil {
		return "nil"
	}
	buf := strings.Builder{}
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err := enc.Encode(node)
	if err != nil {
		return fmt.Sprintf("%s", err)
	}
	return buf.String()
}

// Remove a map key-value pair from node.Content
func RemoveFromMap(node *yaml.Node, name string) error {

	if len(node.Content) == 0 {
		return nil
	}

	idx := -1

	for i, n := range node.Content {
		if n.Value == name {
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("unable to remove %v from map", name)
	}

	newContent := make([]*yaml.Node, 0)
	newContent = append(newContent, node.Content[:idx]...)
	// Remove 2 elements, since the key and value are consecutive elements in the Content array
	newContent = append(newContent, node.Content[idx+2:]...)

	node.Content = newContent

	return nil
}

// Add or replace a map value
func SetMapValue(parent *yaml.Node, name string, val *yaml.Node) {
	found := false
	for i, v := range parent.Content {
		if v.Kind == yaml.ScalarNode && v.Value == name {
			found = true
			parent.Content[i+1] = val
			break
		}
	}
	if !found {
		parent.Content = append(parent.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: name})
		parent.Content = append(parent.Content, val)
	}
}

// Set the value of a sequence element within the node
func SetSequenceValue(parent *yaml.Node, val *yaml.Node, sidx int) {

	if len(parent.Content) <= sidx {
		return
	}

	parent.Content[sidx] = val
}

// Add adds a new scalar property to the map
func Add(m *yaml.Node, name string, val string) {
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: name})
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: val})
}

// AddMap adds a new map to the parent node
// If it already exists, returns the existing map
// If it doesn't exist, returns the new map
func AddMap(parent *yaml.Node, name string) *yaml.Node {
	for i := 0; i < len(parent.Content); i++ {
		if i%2 != 0 {
			continue
		}
		if parent.Content[i].Value == name {
			return parent.Content[i+1]
		}
	}
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: name})
	m := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)}
	parent.Content = append(parent.Content, m)
	return m
}

// YamlVal converts node.Value to a string, int, or bool based on the Tag
func YamlVal(n *yaml.Node) (any, error) {
	var v any
	var err error
	switch n.Tag {
	case "!!bool":
		v, err = strconv.ParseBool(n.Value)
		if err != nil {
			return "", err
		}
	case "!!int":
		v, err = strconv.ParseInt(n.Value, 10, 32)
		if err != nil {
			return "", err
		}
	default:
		v = string(n.Value)
	}
	return v, nil
}

// Diff returns an array of strings describing differences between two nodes
func Diff(node1, node2 *yaml.Node) (diffs []string) {
	switch {
	case node1 == nil && node2 == nil:
		return nil
	case node1 == nil:
		diffs = append(diffs, fmt.Sprintf("Node2: %v", node2.Value))
	case node2 == nil:
		diffs = append(diffs, fmt.Sprintf("Node1: %v", node1.Value))
	default:
		if node1.Kind != node2.Kind {
			diffs = append(diffs, fmt.Sprintf("Node1: %v, Node2: %v", node1.Value, node2.Value))
		} else {
			switch node1.Kind {
			case yaml.MappingNode:
				diffs = appendMappingDiffs(diffs, node1, node2)
			case yaml.SequenceNode:
				diffs = appendSequenceDiffs(diffs, node1, node2)
			case yaml.ScalarNode:
				if node1.Value != node2.Value {
					diffs = append(diffs,
						fmt.Sprintf("Node1: %v, Node2: %v", node1.Value, node2.Value))
				}
			default:
				diffs = append(diffs, fmt.Sprintf("Unsupported node kind: %v", node1.Kind))
			}
		}
	}
	return diffs
}

func appendMappingDiffs(diffs []string, node1, node2 *yaml.Node) []string {
	keys1 := make(map[string]yaml.Node)
	keys2 := make(map[string]yaml.Node)

	for _, child := range node1.Content {
		keys1[child.Value] = *child
	}
	for _, child := range node2.Content {
		keys2[child.Value] = *child
	}

	for k, v := range keys1 {
		if v2, ok := keys2[k]; ok {
			diffs = appendDiffs(diffs, &v, &v2)
			delete(keys2, k)
		} else {
			diffs = append(diffs, fmt.Sprintf("Node1: %v", v.Value))
		}
	}
	for _, v := range keys2 {
		diffs = append(diffs, fmt.Sprintf("Node2: %v", v.Value))
	}
	return diffs
}

func appendSequenceDiffs(diffs []string, node1, node2 *yaml.Node) []string {
	for i := 0; i < len(node1.Content) && i < len(node2.Content); i++ {
		diffs = appendDiffs(diffs, node1.Content[i], node2.Content[i])
	}
	for i := len(node1.Content); i < len(node2.Content); i++ {
		diffs = append(diffs, fmt.Sprintf("Node2: %v", node2.Content[i].Value))
	}
	for i := len(node2.Content); i < len(node1.Content); i++ {
		diffs = append(diffs, fmt.Sprintf("Node1: %v", node1.Content[i].Value))
	}
	return diffs
}

func appendDiffs(diffs []string, node1, node2 *yaml.Node) []string {
	if reflect.DeepEqual(node1, node2) {
		return diffs
	}
	return append(diffs, Diff(node1, node2)...)
}

// MakeMapping returns a pointer to a new yaml mapping node
func MakeMapping() *yaml.Node {
	return &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: make([]*yaml.Node, 0),
	}
}

const (
	Ref    string = "Ref"
	Sub    string = "Fn::Sub"
	GetAtt string = "Fn::GetAtt"
)

func MakeRef(v string) *yaml.Node {
	n := MakeMapping()
	n.Content = append(n.Content, MakeScalar(Ref))
	n.Content = append(n.Content, MakeScalar(v))
	return n
}

func MakeScalar(v string) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: v,
	}
}

func MakeSequence(ss []string) *yaml.Node {
	scalarNodes := make([]*yaml.Node, 0)
	for _, s := range ss {
		scalarNodes = append(scalarNodes, MakeScalar(s))
	}
	return &yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: scalarNodes,
	}
}

// SequenceToStrings converts a sequence of Scalars to a string slice
func SequenceToStrings(n *yaml.Node) []string {
	if n == nil || n.Content == nil {
		return []string{}
	}
	ss := []string{}
	for _, v := range n.Content {
		ss = append(ss, v.Value)
	}
	return ss
}

// DecodeMap converts a node to a string map
func DecodeMap(n *yaml.Node) map[string]any {
	var m map[string]any
	if n != nil {
		decodeErr := n.Decode(&m)
		if decodeErr != nil {
			config.Debugf("decodeMapNode error: %v", decodeErr)
			return m
		}
	}
	return m
}
