package simplify

import (
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/node"
	"gopkg.in/yaml.v3"
)

func mergeComments(comments []string) string {
	out := strings.Builder{}
	for _, c := range comments {
		c := strings.TrimSpace(strings.TrimLeft(c, "# "))
		if c != "" {
			out.WriteString(c)
			out.WriteString(" ")
		}
	}
	return strings.TrimSpace(out.String())
}

// Fix up yaml.Nodes on the way out of a template
func formatNode(n *yaml.Node) *yaml.Node {
	n = node.Clone(n)

	// Is it a map?
	if n.Kind == yaml.MappingNode {
		// Does it have just one key/value pair?
		if len(n.Content) == 2 {
			// Is the key relevant?
			for tag, funcName := range cft.Tags {
				if n.Content[0].Value == funcName {
					// Prepare comments
					headComments := []string{n.HeadComment, n.Content[0].HeadComment, n.Content[1].HeadComment}
					lineComments := []string{n.LineComment, n.Content[0].LineComment, n.Content[1].LineComment}
					footComments := []string{n.FootComment, n.Content[0].FootComment, n.Content[1].FootComment}

					n = n.Content[1]
					n.Tag = tag

					// Is it a GetAtt and is currently a sequence?
					if funcName == "Fn::GetAtt" && n.Kind == yaml.SequenceNode {
						// Are both parts scalars?
						allScalar := true
						parts := make([]string, len(n.Content))
						for i, child := range n.Content {
							if child.Kind != yaml.ScalarNode {
								allScalar = false
								break
							}

							parts[i] = child.Value

							headComments = append(headComments, child.HeadComment)
							lineComments = append(lineComments, child.LineComment)
							footComments = append(footComments, child.FootComment)
						}

						if allScalar {
							n.Content = []*yaml.Node{}
							n.Kind = yaml.ScalarNode
							n.Value = strings.Join(parts, ".")
						}

						n.HeadComment = mergeComments(headComments)
						n.LineComment = mergeComments(lineComments)
						n.FootComment = mergeComments(footComments)
					}

					break
				}
			}
		}
	}

	// Is it a scalar?
	if n.Kind == yaml.ScalarNode {
		// Is it a string
		if n.Tag == "!!str" {
			// Reformat how yaml thinks is best
			if b, err := yaml.Marshal(n.Value); err == nil {
				var newNode yaml.Node
				if err = yaml.Unmarshal(b, &newNode); err == nil {
					n.Style = newNode.Content[0].Style
				}
			}
		}
	}

	for i, child := range n.Content {
		n.Content[i] = formatNode(child)
	}

	return n
}
