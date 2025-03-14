package merge

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
	"gopkg.in/yaml.v3"
)

func checkMerge(name string, dst, src map[string]interface{}) error {
	if _, ok := dst[name]; !ok {
		dst[name] = src[name]
	} else {
		dstMap := dst[name].(map[string]interface{})
		srcMap := src[name].(map[string]interface{})

		for key, value := range srcMap {
			if _, ok := dstMap[key]; ok {
				if forceMerge {
					for i := 2; true; i++ {
						newKey := fmt.Sprintf("%s_%d", key, i)
						if _, ok := dst[newKey]; !ok {
							key = newKey
							break
						}
					}
				} else {
					return fmt.Errorf("templates have clashing %s: %s", name, key)
				}
			}

			dstMap[key] = value
		}
	}

	return nil
}

func mergeTemplates(dstTemplate, srcTemplate *cft.Template) (*cft.Template, error) {
	dst := dstTemplate.Map()
	src := srcTemplate.Map()

	for key, value := range src {
		switch key {
		case "AWSTemplateFormatVersion": // Always overwrite
			dst[key] = value
		case "Description": // Combine descriptions
			if _, ok := dst[key]; !ok {
				dst[key] = src[key]
			} else {
				dst[key] = dst[key].(string) + "\n" + src[key].(string)
			}
		case "Transform": // Append transforms
			if _, ok := dst[key]; !ok {
				dst[key] = src[key]
			} else {
				if _, ok := dst[key].([]interface{}); !ok {
					// Convert to a slice
					dst[key] = []interface{}{dst[key]}
				}

				dst[key] = append(dst[key].([]interface{}), src[key])
			}

		case "Metadata": // Combine metadata
			if _, ok := dst[key]; !ok {
				dst[key] = map[string]interface{}{}
			}

			dstMap, ok := dst[key].(map[string]interface{})
			if !ok {
				return &cft.Template{}, fmt.Errorf("metadata section is not an object (key-value pairs)")
			}
			srcMap, ok := src[key].(map[string]interface{})
			if !ok {
				return &cft.Template{}, fmt.Errorf("metadata section is not an object (key-value pairs)")
			}

			for k := range srcMap {
				if k == "AWS::CloudFormation::Interface" {
					if _, ok := dstMap[k]; !ok {
						dstMap[k] = map[string]interface{}{}
					}

					dstInterface, ok := dstMap[k].(map[string]interface{})
					if !ok {
						return &cft.Template{}, fmt.Errorf("metadata key %s is not an object (key-value pairs)", k)
					}
					srcInterface, ok := srcMap[k].(map[string]interface{})
					if !ok {
						return &cft.Template{}, fmt.Errorf("metadata key %s is not an object (key-value pairs)", k)
					}

					// Concatenate ParameterGroups
					if _, ok := srcInterface["ParameterGroups"]; ok {
						if _, ok := dstInterface["ParameterGroups"]; !ok {
							dstInterface["ParameterGroups"] = []interface{}{}
						}
						dstParameterGroups, ok := dstInterface["ParameterGroups"].([]interface{})
						if !ok {
							return &cft.Template{}, fmt.Errorf("metadata key ParameterGroups is not an array")
						}
						srcParameterGroups, ok := srcInterface["ParameterGroups"].([]interface{})
						if !ok {
							return &cft.Template{}, fmt.Errorf("metadata key ParameterGroups is not an array")
						}

						dstInterface["ParameterGroups"] = append(dstParameterGroups, srcParameterGroups...)
					}

					// Combine ParameterLabels
					if _, ok := srcInterface["ParameterLabels"]; ok {
						if err := checkMerge("ParameterLabels", dstInterface, srcInterface); err != nil {
							return &cft.Template{}, err
						}
					}
					dstMap[k] = dstInterface
				} else {
					if _, ok = dstMap[k]; !ok {
						dstMap[k] = srcMap[k]
					} else {
						if forceMerge {
							for i := 2; true; i++ {
								newKey := fmt.Sprintf("%s_%d", k, i)
								if _, ok := dstMap[newKey]; !ok {
									dstMap[newKey] = srcMap[k]
									break
								}
							}
						} else {
							return &cft.Template{}, fmt.Errorf("templates have clashing %s: %s", key, k)
						}
					}
					dst[key] = dstMap
				}
			}

		default:
			err := checkMerge(key, dst, src)
			if err != nil {
				config.Debugf("key: %v, dst: %v, src: %v", key, dst, src)
				return &cft.Template{}, err
			}
		}
	}

	config.Debugf("map dst: %v", dst)
	retval, err := parse.Map(dst)
	if err != nil {
		return retval, err
	}

	// parse.Map does not actually return a correct cft.Template
	// It's mostly used for unit tests except for here.

	// Add the document node
	docNode := &yaml.Node{Kind: yaml.DocumentNode, Content: make([]*yaml.Node, 0)}
	docNode.Content = append(docNode.Content, retval.Node)
	retval = &cft.Template{Node: docNode}

	// Merge Outputs with Fn::ImportValue
	if mergeImports {
		return mergeOutputImports(retval)
	} else {
		return retval, nil
	}
}
