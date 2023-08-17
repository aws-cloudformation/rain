package simplify

import (
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
)

var valueMap map[string][]string

var isResOutCond bool

var resOutCond = []string{"Resources", "Outputs", "Conditions"}

var finalIsh map[string]modifyMap

//var forEachAdd []modifyMap

func modifyTemplate(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.DocumentNode {
		node = node.Content[0]
	}

	keyValueMap := modifyNode(node, orders)

	return keyValueMap
}

type nodeMap0 struct {
	keys   []*yaml.Node
	keyMap map[string]*yaml.Node
	values map[string]*yaml.Node
}

type modifyMap struct {
	key    string
	list   []string
	values map[string]modifyMap
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

func addForEach(m modifyMap, forEachAdd []modifyMap, key string) modifyMap {
	if m.key == "" {
		for key1, value1 := range m.values {
			if len(forEachAdd) <= 0 {
				forEachAdd = append(forEachAdd, modifyMap{key: "Variable" + strconv.Itoa(len(forEachAdd)), values: value1.values})
				delete(m.values, key1)
			} else {
				for key2, value2 := range value1.values {
					for index, element := range forEachAdd {
						val, ok := element.values[key2]
						if ok {
							if val.values != nil {
								element = compareValue(key2, value2, val, forEachAdd)
								if element.key == "REMOVE" {
									break
								}
								for _, elem := range element.list {
									forEachAdd[index].list = append(forEachAdd[index].list, elem)
								}
								delete(m.values, key1)
								break
							} else if val.key != value2.key {
								element.list = append(element.list, val.key, value2.key)
								element.values[key2] = modifyMap{key: "Ref: Variable" + strconv.Itoa(index)}
								delete(m.values, key1)
							}
						} else {
							break
						}
					}
				}
			}
		}
	} else {
		return m
	}

	for i := 0; i < len(forEachAdd); i++ {
		if forEachAdd[i].list != nil {
			tempVal := forEachAdd[i].values
			forEachAdd[i].values = make(map[string]modifyMap)
			forEachAdd[i].values[strings.TrimSuffix(key, "s")+"${"+forEachAdd[i].key+"}"] = modifyMap{values: tempVal}
			m.values["Fn::ForEach::Loop"+strconv.Itoa(i)] = forEachAdd[i]
		} else {

		}
	}

	return m
}

func modifyNode(node *yaml.Node, order ordering) *yaml.Node {
	if node == nil {
		return nil
	}

	if node.Kind != yaml.MappingNode {
		return node
	}

	m := createMap(node)
	forEachAdd := []modifyMap{}
	for key, value := range m {
		if checkIfAddForEach(key) {
			m[key] = addForEach(value, forEachAdd, key)
		}
	}

	// Build the output node
	out := &yaml.Node{
		Kind: yaml.MappingNode,
	}

	for key, value := range m {
		node = addToTemplate(&yaml.Node{Kind: yaml.MappingNode}, key, value, 1, 1)
		for int := range node.Content {
			out.Content = append(out.Content, node.Content[int])
		}
	}

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
		//Tag:  "!!map",
		Line: line,
	}

	if key != "" {
		tempKeyNode.Value = key
		out.Content = append(out.Content, tempKeyNode)
	} else if value.key != "" {
		tempKeyNode.Value = value.key
		out.Content = append(out.Content, tempKeyNode)
	}
	if value.values != nil {
		for key1, value1 := range value.values {
			tempValueNode.Column = tempKeyNode.Column + 1
			tempValueNode.Line = line + 1
			newContent := addToTemplate(&yaml.Node{Kind: yaml.MappingNode}, key1, value1, line+1, tempKeyNode.Column)
			if len(value1.list) > 0 {
				tempValueNode.Kind = yaml.MappingNode
				tempValueNode.Content = append(tempValueNode.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Line: line + 1, Column: tempKeyNode.Column + 1, Value: key1})
				tempValueNode.Content = append(tempValueNode.Content, &yaml.Node{Kind: yaml.SequenceNode, Line: line + 1, Column: tempKeyNode.Column + 1})
				tempValueNode.Content[1].Content = append(tempValueNode.Content[1].Content, &yaml.Node{Kind: yaml.ScalarNode, Line: line + 1, Value: value1.key})
				tempValueNode.Content[1].Content = append(tempValueNode.Content[1].Content, &yaml.Node{Kind: yaml.SequenceNode, Line: line + 1, Column: tempKeyNode.Column + 1})
				for _, elem := range value1.list {
					tempValueNode.Content[1].Content[1].Content = append(tempValueNode.Content[1].Content[1].Content, &yaml.Node{Kind: yaml.ScalarNode, Value: elem})
				}
				tempValueNode.Content[1].Content = append(tempValueNode.Content[1].Content, &yaml.Node{Kind: yaml.MappingNode, Line: line + 1, Column: tempKeyNode.Column + 1})
				for elem2, value2 := range value1.values {
					//tempValueNode.Content[1].Content[2].Content = append(tempValueNode.Content[1].Content[2].Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Line: line + 1, Column: tempKeyNode.Column + 1, Value: elem2})
					tempValueNode.Content[1].Content[2].Content = append(tempValueNode.Content[1].Content[2].Content, &yaml.Node{Kind: yaml.ScalarNode, Line: line + 1, Column: tempKeyNode.Column + 1, Value: elem2})
					tempValueNode.Content[1].Content[2].Content = append(tempValueNode.Content[1].Content[2].Content, &yaml.Node{Kind: yaml.MappingNode, Line: line + 1, Column: tempKeyNode.Column + 1})
					for elem3, value3 := range value2.values {
						print(value3.key)
						//tempValueNode.Content[1].Content[2].Content[1].Content = append(tempValueNode.Content[1].Content[2].Content, value2)
						newContent := addToTemplate(&yaml.Node{Kind: yaml.MappingNode}, elem3, value3, line+1, tempKeyNode.Column)
						for _, value4 := range newContent.Content {
							tempValueNode.Content[1].Content[2].Content[1].Content = append(tempValueNode.Content[1].Content[2].Content[1].Content, value4)
						}
					}
				}
			} else {
				if newContent.Content[0].Value == "" {
					tempValueNode.Kind = yaml.SequenceNode
				} else {
					tempValueNode.Kind = yaml.MappingNode
				}
				for _, value2 := range newContent.Content {
					tempValueNode.Content = append(tempValueNode.Content, value2)
				}
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
