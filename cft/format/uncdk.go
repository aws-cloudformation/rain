package format

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/visitor"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

func UnCDK(t *cft.Template) error {

	// Remove these nodes:
	//
	// Resources:
	//   CDKMetadata:
	//   {*}:
	//     Metadata:
	//       aws:cdk:path:
	//       aws:asset:path:
	//       aws:asset:property:
	// Conditions:
	//   CDKMetadataAvailable:
	// Parameters:
	//   BootstrapVersion:
	// Rules:
	//   CheckBootstrapVersion:

	removals := make(map[string][]string)
	removals[string(cft.Resources)] = []string{"CDKMetadata"}
	removals[string(cft.Conditions)] = []string{"CDKMetadataAvailable"}
	removals[string(cft.Parameters)] = []string{"BootstrapVersion"}
	removals[string(cft.Rules)] = []string{"CheckBootstrapVersion"}

	for k, v := range removals {
		section, err := t.GetSection(cft.Section(k))
		if err != nil {
			continue // Section not found
		}
		for _, name := range v {
			n := s11n.GetMap(section, name)
			if n != nil {
				node.RemoveFromMap(section, name)
			}
		}
	}

	// Iterate through all the resources to remove cdk metadata,
	// And fix the logical ids so they are easier to read

	resources, err := t.GetSection(cft.Resources)
	if err != nil {
		return err
	}

	commonPrefix := getCommonResourcePrefix(t)

	// Store the resource logical id node each time we see a repeated name
	// Start without a number, for example "Bucket"
	// If we see another one, fix the first one to be "Bucket0"
	allNames := make(map[string][]*yaml.Node)

	for i := 0; i < len(resources.Content); i += 1 {
		if i%2 != 0 {
			continue
		}
		logicalId := resources.Content[i].Value
		resource := resources.Content[i+1]

		// Simplify the logical id
		_, typ, _ := s11n.GetMapValue(resource, "Type")
		if typ == nil {
			return fmt.Errorf("expected %s to have Type", logicalId)
		}
		oldName := resources.Content[i].Value
		newName := createNewName(typ.Value, logicalId, commonPrefix)
		if nameNodes, ok := allNames[newName]; ok {
			// We've seen this one before
			nameNodes = append(nameNodes, resources.Content[i])
			allNames[newName] = nameNodes
			for nodeIdx, node := range nameNodes {
				sequential := fmt.Sprintf("%s%d", newName, nodeIdx)
				priorValue := node.Value
				node.Value = sequential
				replaceNames(t, priorValue, sequential)
			}
		} else {
			// We haven't seen this name yet
			resources.Content[i].Value = newName
			allNames[newName] = make([]*yaml.Node, 0)
			allNames[newName] = append(allNames[newName], resources.Content[i])
			replaceNames(t, oldName, newName)
		}

		// Remove the cdk path and asset metadata
		_, metadata, _ := s11n.GetMapValue(resource, string(cft.Metadata))
		if metadata != nil {
			stringsToRemove := []string{
				"aws:cdk:path",
				"aws:asset:path",
				"aws:asset:property",
				"aws:asset:is-bundled",
				"cfn_nag",
			}
			for _, s := range stringsToRemove {
				node.RemoveFromMap(metadata, s)
			}
			// If the resource Metadata node is empty, remove it
			if len(metadata.Content) == 0 {
				node.RemoveFromMap(resource, string(cft.Metadata))
			}
		}
	}

	// Remove any empty sections
	t.RemoveEmptySections()

	// Replace Joins with Subs to make them easier to read
	joinToSub(t)

	return nil // TODO

}

func joinSeqToString(seq *yaml.Node) string {
	if len(seq.Content) != 2 {
		return "Invalid Join"
	}
	j := seq.Content[0].Value
	tokens := seq.Content[1]
	retval := ""
	for i, token := range tokens.Content {
		if i != 0 {
			retval += j
		}
		if token.Kind == yaml.ScalarNode {
			retval += token.Value
		}
		if token.Kind == yaml.MappingNode {
			if token.Content[0].Value == "Ref" {
				retval += "${" + token.Content[1].Value + "}"
			}
			if token.Content[0].Value == "Fn::GetAtt" {
				retval += "${" + token.Content[1].Content[0].Value
				retval += "." + token.Content[1].Content[1].Value + "}"
			}
		}
	}
	return retval
}

func joinToSub(t *cft.Template) {
	vf := func(n *visitor.Visitor) {
		yamlNode := n.GetYamlNode()
		if yamlNode.Kind == yaml.MappingNode {
			if len(yamlNode.Content) == 2 && yamlNode.Content[0].Value == "Fn::Join" {
				seq := yamlNode.Content[1]
				if seq.Kind == yaml.SequenceNode {
					yamlNode.Content[0].Value = "Fn::Sub"
					yamlNode.Content[1].Value = joinSeqToString(seq)
					yamlNode.Content[1].Kind = yaml.ScalarNode
					yamlNode.Content[1].Content = make([]*yaml.Node, 0)
				}
			}
		}
	}
	visitor := visitor.NewVisitor(t.Node)
	visitor.Visit(vf)

}

func replaceNames(t *cft.Template, oldName, newName string) {
	vf := func(n *visitor.Visitor) {
		yamlNode := n.GetYamlNode()
		if yamlNode.Kind == yaml.ScalarNode {
			if yamlNode.Value == oldName {
				yamlNode.Value = newName
			}
		}
	}
	visitor := visitor.NewVisitor(t.Node)
	visitor.Visit(vf)
}

// getCommonTemplatePrefix attempts to find a common string that begins all resource names.
func getCommonResourcePrefix(t *cft.Template) string {
	resources, err := t.GetSection(cft.Resources)
	if err != nil {
		return ""
	}
	logicalIds := make([]string, 0)
	for i := 0; i < len(resources.Content); i += 2 {
		logicalId := resources.Content[i].Value
		logicalIds = append(logicalIds, logicalId)
	}
	return getCommonPrefix(logicalIds)
}

// getCommonPrefix attempts to find a common string that begins all strings in the slice.
func getCommonPrefix(logicalIds []string) string {
	if len(logicalIds) < 2 {
		return ""
	}
	retval := ""
	prefixes := make([]string, 0)
	for j := 1; j < len(logicalIds); j++ {
		prefix := ""
		for i, c := range logicalIds[0] {
			second := []rune(logicalIds[j])
			if len(second) > i && second[i] == c {
				prefix += string(c)
			} else {
				prefixes = append(prefixes, prefix)
				if retval == "" {
					retval = prefix
				}
				for _, p := range prefixes {
					// Pick the shortest prefix
					if len(p) < len(retval) && retval != "" {
						retval = p
					}
				}
				break
			}
		}
	}

	common := true
	for _, id := range logicalIds {
		if !strings.HasPrefix(id, retval) {
			common = false
			break
		}
	}
	if common {
		return retval
	}
	return ""
}

// stripSuffix attempts to remove the random 8 characters at the end of ids
func stripSuffix(s string) string {

	if len(s) <= 8 {
		return s
	}

	// Too simple. For imported constructs, you can end up with several
	// Strip off the random 8 digit string at the end
	//return newName[:len(newName)-8]

	suffixLen := 0

	for i := len(s) - 1; i >= 0; i-- {
		isUpper := unicode.IsUpper(rune(s[i])) && unicode.IsLetter(rune(s[i]))
		isDigit := unicode.IsDigit(rune(s[i]))
		if isUpper || isDigit {
			suffixLen += 1
		} else {
			break
		}
	}

	if suffixLen == len(s) {
		return s
	}

	// Round to the nearest 8 in case a name ended with a capital letter or number
	suffixLen = suffixLen - (suffixLen % 8)

	return s[:len(s)-suffixLen]
}

// createNewName converts the cdk generated name into something that is easier to read.
func createNewName(typeName string, logicalId string, commonPrefix string) string {
	newName := ""
	if commonPrefix != "" {
		newName = strings.Replace(logicalId, commonPrefix, "", -1)
		return stripSuffix(newName)
	}
	tokens := strings.Split(typeName, "::")
	if len(tokens) == 3 {
		newName = tokens[2]
	} else if len(tokens) == 2 && tokens[0] == "Custom" {
		newName = strings.Replace(tokens[1], "-", "", -1)
	} else {
		newName = strings.Replace(typeName, "::", "", -1)
	}
	return newName
}
