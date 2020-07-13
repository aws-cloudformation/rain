package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cfn"
	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/aws-cloudformation/rain/cfn/parse"
	"github.com/spf13/cobra"
)

var forceMerge = false

func checkMerge(name string, dst cfn.Template, src cfn.Template) {
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
					panic(fmt.Errorf("Templates have clashing %s: %s", name, key))
				}
			}

			dstMap[key] = value
		}
	}
}

func mergeTemplates(dst cfn.Template, src cfn.Template) {
	for key, value := range src.Map() {
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
		default:
			checkMerge(key, dst, src)
		}
	}
}

var mergeCmd = &cobra.Command{
	Use:                   "merge <template> <template> ...",
	Short:                 "Merge two or more CloudFormation templates",
	Long:                  "Merges all specified CloudFormation templates, print the resultant template to standard out",
	Args:                  cobra.MinimumNArgs(2),
	Annotations:           templateAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		templates := make([]cfn.Template, len(args))

		for i, fn := range args {
			templates[i], err = parse.File(fn)
			if err != nil {
				panic(fmt.Errorf("Unable to open template '%s': %s", fn, err))
			}
		}

		var merged cfn.Template

		for i, template := range templates {
			if i == 0 {
				merged = template
				continue
			}

			mergeTemplates(merged, template)
		}

		fmt.Println(format.Template(merged, format.Options{}))
	},
}

func init() {
	mergeCmd.Flags().BoolVarP(&forceMerge, "force", "f", false, "Don't warn on clashing attributes; rename them instead. Note: this will not rename Refs, GetAtts, etc.")
	Rain.AddCommand(mergeCmd)
}
