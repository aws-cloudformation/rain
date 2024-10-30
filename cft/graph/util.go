package graph

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
)

func getRefs(t map[string]interface{}) []string {
	return findRefs(t)
}

func parseSubString(refs []string, substr string) []string {
	words, err := parse.ParseSub(substr, false)
	if err != nil {
		config.Debugf("Unable to parse Sub %s: %v", substr, err)
		return refs
	}

	for _, word := range words {
		switch word.T {
		case parse.AWS:
			refs = append(refs, fmt.Sprintf("AWS::%s", word.W))
		case parse.REF:
			refs = append(refs, word.W)
		case parse.GETATT:
			left, _, found := strings.Cut(word.W, ".")
			if !found {
				config.Debugf("unexpected GetAtt %s", word.W)
			} else {
				refs = append(refs, left)
			}
		}
	}

	config.Debugf("After parsing Sub, refs is now: %v", refs)

	return refs
}

func findRefs(t map[string]interface{}) []string {
	refs := make([]string, 0)

	for key, value := range t {
		switch key {
		case "DependsOn":
			switch v := value.(type) {
			case string:
				refs = append(refs, v)
			case []interface{}:
				for _, d := range v {
					refs = append(refs, d.(string))
				}
			default:
				config.Debugf("invalid DependsOn: %v, %v", key, value)
			}
		case "Ref":
			refs = append(refs, value.(string))
		case "Fn::GetAtt":
			switch v := value.(type) {
			case string:
				parts := strings.Split(v, ".")
				refs = append(refs, parts[0])
			case []interface{}:
				if s, ok := v[0].(string); ok {
					refs = append(refs, s)
				}
			default:
				fmt.Printf("Malformed GetAtt: %T\n", v)
			}
		case "Fn::Sub":
			switch v := value.(type) {
			case string:
				refs = parseSubString(refs, v)
			case []interface{}:
				switch {
				case len(v) != 2:
					fmt.Printf("Malformed Sub: %T\n", v)
				default:
					switch parts := v[1].(type) {
					case map[string]interface{}:
						for _, part := range parts {
							switch p := part.(type) {
							case map[string]interface{}:
								refs = append(refs, findRefs(p)...)
							default:
								fmt.Printf("Malformed Sub: %T\n", v)
							}
						}
					default:
						fmt.Printf("Malformed Sub: %T\n", v)
					}
				}
			default:
				fmt.Printf("Malformed Sub: %T\n", v)
			}
		default:
			for _, tree := range findTrees(value) {
				refs = append(refs, findRefs(tree)...)
			}
		}
	}

	return refs
}

func findTrees(value interface{}) []map[string]interface{} {
	trees := make([]map[string]interface{}, 0)

	switch v := value.(type) {
	case map[string]interface{}:
		trees = append(trees, v)
	case []interface{}:
		for _, child := range v {
			trees = append(trees, findTrees(child)...)
		}
	}

	return trees
}
