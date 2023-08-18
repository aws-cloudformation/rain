//

package simplify

import (
	"gopkg.in/yaml.v3"
	"sort"
	"strconv"
	"strings"
)

var resOutCond = []string{"Resources", "Outputs", "Conditions"}

func modifyTemplate(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.DocumentNode {
		node = node.Content[0]
	}

	keyValueMap := modifyNode(node)

	return keyValueMap
}

type modifyMap struct {
	key    string
	list   []string
	values map[string]modifyMap
}

func sortKeys(templateMap map[string]modifyMap) []string {
	keys := make([]string, 0, len(templateMap))

	for k := range templateMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}

func createMap(node *yaml.Node) map[string]modifyMap {
	valueMap := make(map[string]modifyMap)

	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		var value *yaml.Node
		if i != len(node.Content)-1 {
			value = node.Content[i+1]
		}

		if value == nil {
			valueMap[key.Value] = modifyMap{values: createMap(node.Content[i])}
		} else if value.Value == "" {
			valueMap[key.Value] = modifyMap{values: createMap(node.Content[i+1])}
		} else {
			valueMap[key.Value] = modifyMap{key: value.Value}
		}
	}

	return valueMap
}

func checkIfAddForEach(key string) bool {
	for _, value := range resOutCond {
		if key == value {
			return true
		}
	}
	return false
}

// Compares the values in the parameters to the parameter values in Fn::ForEach
func compareValue(key string, value modifyMap, forEachVal modifyMap, forEachAdd []modifyMap) modifyMap {
	for index := range forEachAdd {
		if forEachVal.key == "" {
			for k1, v1 := range forEachVal.values {
				if k1 != "" {
					for k2, v2 := range value.values {
						_, ok := value.values[k1]
						if ok {
							if v1.key != v2.key && k1 == k2 {
								if v1.key != "Ref: Variable"+strconv.Itoa(index) {
									forEachVal.list = append(forEachVal.list, v1.key, v2.key)
									forEachVal.values[k1] = modifyMap{key: "Ref: Variable" + strconv.Itoa(index)}
								} else {
									forEachVal.list = append(forEachVal.list, v2.key)
								}
							}
						} else {
							forEachVal.key = "REMOVE"
							return forEachVal
						}
					}
				} else {
					forEachVal = compareValue(key, value, forEachVal.values[k1], forEachAdd)
				}
			}
		}
	}
	return forEachVal
}

// Create a Fn::forEach function
func addForEach(templateMap modifyMap, forEachAdd []modifyMap, key string) modifyMap {
	if templateMap.key == "" {
		for key1, value1 := range templateMap.values {
			// the first resource/output/condition is added to the Fn::ForEach function and the parameters are added
			// to the OutputKey and OutputValue
			if len(forEachAdd) <= 0 {
				forEachAdd = append(forEachAdd, modifyMap{key: "Variable" + strconv.Itoa(len(forEachAdd)), values: value1.values})
				delete(templateMap.values, key1)
			} else {
				for key2, value2 := range value1.values {
					for index, parameter := range forEachAdd {
						val, ok := parameter.values[key2]
						if ok {
							if val.values != nil {
								parameter = compareValue(key2, value2, val, forEachAdd)
								// parameter has been added to Fn::ForEach, so it will be removed
								if parameter.key == "REMOVE" {
									break
								}
								forEachAdd[index].list = append(forEachAdd[index].list, parameter.list...)
								delete(templateMap.values, key1)
								break
							} else if val.key != value2.key {
								parameter.list = append(parameter.list, val.key, value2.key)
								parameter.values[key2] = modifyMap{key: "Ref: Variable" + strconv.Itoa(index)}
								delete(templateMap.values, key1)
							}
						} else {
							break
						}
					}
				}
			}
		}
	} else {
		return templateMap
	}

	// Modify Fn::ForEach OutputKey
	for i := 0; i < len(forEachAdd); i++ {
		if forEachAdd[i].list != nil {
			tempVal := forEachAdd[i].values
			forEachAdd[i].values = make(map[string]modifyMap)
			forEachAdd[i].values[strings.TrimSuffix(key, "s")+"${"+forEachAdd[i].key+"}"] = modifyMap{values: tempVal}
			templateMap.values["Fn::ForEach::Loop"+strconv.Itoa(i)] = forEachAdd[i]
		}
	}

	return templateMap
}

func modifyNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	if node.Kind != yaml.MappingNode {
		return node
	}

	templateMap := createMap(node)
	forEachAdd := []modifyMap{}
	for key, value := range templateMap {
		if checkIfAddForEach(key) {
			templateMap[key] = addForEach(value, forEachAdd, key)
		}
	}

	// Build the output node
	out := &yaml.Node{
		Kind: yaml.MappingNode,
	}

	keys := sortKeys(templateMap)

	for _, k := range keys {
		node = addToTemplate(&yaml.Node{Kind: yaml.MappingNode}, k, templateMap[k], 1, 1)
		out.Content = append(out.Content, node.Content...)
	}

	out = orderTemplate(out)

	return out
}

func addToTemplate(out *yaml.Node, key string, value modifyMap, line int, column int) *yaml.Node {
	tempKeyNode := &yaml.Node{
		Kind:   yaml.ScalarNode,
		Tag:    "!!str",
		Line:   line,
		Column: column,
	}
	tempValueNode := &yaml.Node{
		Line: line,
	}

	if key != "" {
		tempKeyNode.Value = key
		out.Content = append(out.Content, tempKeyNode)
	} else if value.key != "" {
		tempKeyNode.Value = value.key
		out.Content = append(out.Content, tempKeyNode)
	}

	// Add map to yaml.Node
	if value.values != nil {
		keys1 := sortKeys(value.values)
		for _, key1 := range keys1 {
			tempValueNode.Column = tempKeyNode.Column + 1
			tempValueNode.Line = line + 1
			newContent := addToTemplate(&yaml.Node{Kind: yaml.MappingNode}, key1, value.values[key1], line+1, tempKeyNode.Column)
			// Adding Fn::ForEach to yaml.Node
			if len(value.values[key1].list) > 0 {
				tempValueNode.Kind = yaml.MappingNode
				tempValueNode.Content = append(tempValueNode.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str",
					Line: line + 1, Column: tempValueNode.Column, Value: key1}) // Fn::ForEach Loop Name
				tempValueNode.Content = append(tempValueNode.Content, &yaml.Node{Kind: yaml.SequenceNode,
					Line: line + 1, Column: tempValueNode.Column}) // body of Fn::ForEach
				tempValueNode.Content[1].Content = append(tempValueNode.Content[1].Content,
					&yaml.Node{Kind: yaml.ScalarNode, Line: line + 1, Value: value.values[key1].key}) // Fn::ForEach Identifier
				tempValueNode.Content[1].Content = append(tempValueNode.Content[1].Content,
					&yaml.Node{Kind: yaml.SequenceNode, Line: line + 1, Column: tempKeyNode.Column + 1}) // Fn::ForEach Collection
				// Adding values to Collection
				sort.Strings(value.values[key1].list)
				for _, elem := range value.values[key1].list {
					tempValueNode.Content[1].Content[1].Content = append(tempValueNode.Content[1].Content[1].Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: elem})
				}
				tempValueNode.Content[1].Content = append(tempValueNode.Content[1].Content,
					&yaml.Node{Kind: yaml.MappingNode, Line: line + 1, Column: tempKeyNode.Column + 1}) // OutputKey and OutputValue Map
				// Adding OutputKey and OutputValue
				for elem2, value2 := range value.values[key1].values {
					tempValueNode.Content[1].Content[2].Content = append(tempValueNode.Content[1].Content[2].Content,
						&yaml.Node{Kind: yaml.ScalarNode, Line: line + 1, Column: tempKeyNode.Column + 1, Value: elem2}) // OutputKey
					tempValueNode.Content[1].Content[2].Content = append(tempValueNode.Content[1].Content[2].Content,
						&yaml.Node{Kind: yaml.MappingNode, Line: line + 1, Column: tempKeyNode.Column + 1}) // OutputValue Map

					keys := sortKeys(value2.values)

					// Adding OutputValues
					for _, k := range keys {
						newContent := addToTemplate(&yaml.Node{Kind: yaml.MappingNode}, k, value2.values[k], line+1, tempKeyNode.Column)
						tempValueNode.Content[1].Content[2].Content[1].Content = append(tempValueNode.Content[1].Content[2].Content[1].Content, newContent.Content...)
					}
				}
			} else {
				if newContent.Content[0].Value == "" {
					tempValueNode.Kind = yaml.SequenceNode
				} else {
					tempValueNode.Kind = yaml.MappingNode
				}
				tempValueNode.Content = append(tempValueNode.Content, newContent.Content...)
			}
		}
	} else {
		if strings.Contains(value.key, "Ref: ") {
			value.key = strings.Replace(value.key, "Ref: ", "", 1)
			tempValueNode.Tag = "!Ref"
		} else {
			tempValueNode.Tag = "!!str"
		}
		tempValueNode.Value = value.key
		tempValueNode.Column = column + len(tempKeyNode.Value) + 2
		tempValueNode.Line = line
		tempValueNode.Kind = yaml.ScalarNode
	}

	out.Content = append(out.Content, tempValueNode)

	return out
}
