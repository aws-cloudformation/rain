package node

import (
	"gopkg.in/yaml.v3"
)

// MergeNodes merges two mapping nodes, replacing original values found in override.
// Objects, which are yaml Mapping nodes, are merged together so that identical keys
// are replaced by what is in original. Lists, which are yaml Sequences, are merged
// such that if the list contains a Mapping, and that Mapping has any identical keys,
// they are replaced instead of appended.
func MergeNodes(original *yaml.Node, override *yaml.Node) *yaml.Node {

	retval := &yaml.Node{Kind: override.Kind, Content: make([]*yaml.Node, 0)}

	// If the nodes are not the same kind, just return the override
	if override.Kind != original.Kind {
		return override
	}

	if override.Kind == yaml.ScalarNode {
		return override
	}

	if override.Kind == yaml.SequenceNode {
		// Use the recursive sequence merging function
		retval = mergeSeq(original, override)
		return retval
	}

	// else they are both Mapping nodes

	// Start by adding everything in the override node,
	// merging if the same key is found in original
	for i := 0; i < len(override.Content); i += 2 {
		found := false
		for j := 0; j < len(original.Content); j += 2 {
			if override.Content[i].Value == original.Content[j].Value {
				// Merge the matching nodes
				retval.Content = append(retval.Content,
					MakeScalar(override.Content[i].Value))
				retval.Content = append(retval.Content,
					MergeNodes(original.Content[j+1], override.Content[i+1]))
				found = true
				break
			}
		}
		if !found {
			// Add a clone of the override
			retval.Content = append(retval.Content,
				MakeScalar(override.Content[i].Value))
			retval.Content = append(retval.Content,
				Clone(override.Content[i+1]))
		}
	}

	// Add anything from the original that is not in the override
	for j := 0; j < len(original.Content); j += 2 {
		found := false
		for i := 0; i < len(override.Content); i += 2 {
			if original.Content[j].Value == override.Content[i].Value {
				found = true
				break
			}
		}
		if !found {
			retval.Content = append(retval.Content,
				MakeScalar(original.Content[j].Value))
			retval.Content = append(retval.Content,
				Clone(original.Content[j+1]))
		}
	}

	return retval
}

// mergeSeq handles merging of sequences that might be nested within mapping nodes
func mergeSeq(original *yaml.Node, override *yaml.Node) *yaml.Node {
	// If either node is nil, return the other one
	if original == nil {
		return Clone(override)
	}
	if override == nil {
		return Clone(original)
	}

	// If nodes are not both sequences, return override
	if original.Kind != yaml.SequenceNode || override.Kind != yaml.SequenceNode {
		return Clone(override)
	}

	// Create result sequence
	result := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: make([]*yaml.Node, 0),
	}

	// First add all items from original
	for _, origItem := range original.Content {
		result.Content = append(result.Content, Clone(origItem))
	}

	// Then process override items
	for _, overrideItem := range override.Content {
		if overrideItem.Kind != yaml.MappingNode {
			// For non-mapping nodes, just append
			result.Content = append(result.Content, Clone(overrideItem))
			continue
		}

		// For mapping nodes, try to find a matching item to merge with
		matched := false
		for i, existingItem := range result.Content {
			if existingItem.Kind != yaml.MappingNode {
				continue
			}

			// Check if the mapping nodes have matching key-value pairs
			if haveSameKeyValue(existingItem, overrideItem) {
				// Found a match, merge them
				result.Content[i] = MergeNodes(existingItem, overrideItem)
				matched = true
				break
			}
		}

		// If no match found, append as new item
		if !matched {
			result.Content = append(result.Content, Clone(overrideItem))
		}
	}

	return result
}

// haveSameKeyValue checks if two mapping nodes have at least one matching key-value pair
func haveSameKeyValue(node1, node2 *yaml.Node) bool {
	if node1.Kind != yaml.MappingNode || node2.Kind != yaml.MappingNode {
		return false
	}

	// Compare each key-value pair from node1 with node2
	for i := 0; i < len(node1.Content); i += 2 {
		key1 := node1.Content[i].Value
		val1 := node1.Content[i+1].Value

		// Look for matching pair in node2
		for j := 0; j < len(node2.Content); j += 2 {
			key2 := node2.Content[j].Value
			val2 := node2.Content[j+1].Value

			if key1 == key2 && val1 == val2 {
				return true
			}
		}
	}

	return false
}
