package merge

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
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

func mergeTemplates(dstTemplate, srcTemplate cft.Template) (cft.Template, error) {
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
			dstMap := dst[key].(map[string]interface{})
			srcMap := src[key].(map[string]interface{})
			for k, _ := range srcMap {
				if _, ok := dstMap[k]; !ok {
					dstMap[k] = srcMap[k]
				} else {
					if k == "AWS::CloudFormation::Interface" {
						dstInterface := dstMap[k].(map[string]interface{})
						srcInterface := srcMap[k].(map[string]interface{})

						// Concatenate ParameterGroups
						if srcParameterGroups, ok := srcInterface["ParameterGroups"].([]interface{}); ok {
							dstParameterGroups, ok := dstInterface["ParameterGroups"].([]interface{})
							if !ok {
								dstParameterGroups = srcParameterGroups
							} else {
								dstParameterGroups = append(dstParameterGroups, srcParameterGroups...)
							}
							dstInterface["ParameterGroups"] = dstParameterGroups
						}
						// Combine ParameterLabels
						if _, ok := srcInterface["ParameterLabels"]; ok {
							if err := checkMerge("ParameterLabels", dstInterface, srcInterface); err != nil {
								return cft.Template{}, err
							}
						}
					} else {
						if forceMerge {
							for i := 2; true; i++ {
								newKey := fmt.Sprintf("%s_%d", k, i)
								if _, ok := dstMap[newKey]; !ok {
									k = newKey
									break
								}
							}
						} else {
							return cft.Template{}, fmt.Errorf("templates have clashing %s: %s", key, k)
						}
						dstMap[k] = value
					}
				}
			}

		default:
			err := checkMerge(key, dst, src)
			if err != nil {
				return cft.Template{}, err
			}
		}
	}

	return parse.Map(dst)
}
