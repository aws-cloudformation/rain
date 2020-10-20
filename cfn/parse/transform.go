package parse

import (
	"strings"

	"gopkg.in/yaml.v3"
)

var tags = []string{
	"And",
	"Base64",
	"Cidr",
	"Equals",
	"FindInMap",
	"GetAZs",
	"GetAtt",
	"If",
	"ImportValue",
	"Join",
	"Not",
	"Or",
	"Ref",
	"Select",
	"Split",
	"Sub",
	"Transform",
}

func transform(node *yaml.Node) {
	// Fix badly-parsed numbers
	if node.ShortTag() == "!!float" && node.Value[0] == '0' {
		node.Tag = "!!str"
	}

	// Fix badly-parsed timestamps which are often used for versions in cloudformation
	if node.ShortTag() == "!!timestamp" {
		node.Tag = "!!str"
	}

	// See if we're dealing with a Fn:: tag
	for _, tag := range tags {
		if node.ShortTag() == "!"+tag {
			key := tag
			if tag != "Ref" && tag != "Condition" {
				key = "Fn::" + key
			}

			body := yaml.Node{
				Kind:        node.Kind,
				Style:       0,
				Value:       node.Value,
				Content:     node.Content,
				HeadComment: node.HeadComment,
				LineComment: node.LineComment,
				FootComment: node.FootComment,
				Line:        node.Line,
				Column:      node.Column,
			}

			body.Tag = body.ShortTag()

			if tag == "GetAtt" && body.Tag == "!!str" {
				body.Kind = yaml.SequenceNode

				parts := strings.SplitN(node.Value, ".", 2)

				body.Content = []*yaml.Node{
					&yaml.Node{
						Kind:  yaml.ScalarNode,
						Style: 0,
						Tag:   "!!str",
						Value: parts[0],
					},
					&yaml.Node{
						Kind:  yaml.ScalarNode,
						Style: 0,
						Tag:   "!!str",
						Value: parts[1],
					},
				}
			}

			node.Kind = yaml.MappingNode
			node.Style = 0
			node.Tag = "!!map"
			node.Content = []*yaml.Node{
				&yaml.Node{
					Kind:  yaml.ScalarNode,
					Style: 0,
					Tag:   "!!str",
					Value: key,
				},
				&body,
			}
		}
	}

	for _, child := range node.Content {
		transform(child)
	}
}
