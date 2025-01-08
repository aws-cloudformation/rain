package format

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/visitor"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

func UnCDK(t cft.Template) error {

	// Remove these nodes:
	//
	// Resources:
	//   CDKMetadata:
	//   {*}:
	//     Metadata:
	//       aws:cdk:path:
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
		tokens := strings.Split(typ.Value, "::")
		if len(tokens) < 3 {
			// TODO ::Module would break here
			config.Debugf("unexpected %s Type is %s", logicalId, typ.Value)
		} else {
			oldName := resources.Content[i].Value
			newName := tokens[2]
			if nameNodes, ok := allNames[newName]; ok {
				// We've seen this one before
				nameNodes = append(nameNodes, resources.Content[i])
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
		}

		// Remove the cdk path
		_, metadata, _ := s11n.GetMapValue(resource, string(cft.Metadata))
		if metadata != nil {
			cdkPath := "aws:cdk:path"
			found := false
			for m := 0; m < len(metadata.Content); m += 2 {
				if metadata.Content[m].Value == cdkPath {
					found = true
					break
				}
			}
			if found {
				err := node.RemoveFromMap(metadata, cdkPath)
				if err != nil {
					return err // This should not happen
				}
			}
		}
		// If the resource Metadata node is empty, remove it
		if len(metadata.Content) == 0 {
			node.RemoveFromMap(resource, string(cft.Metadata))
		}
	}

	// Remove any empty sections
	t.RemoveEmptySections()

	return nil // TODO

}

func replaceNames(t cft.Template, oldName, newName string) {
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
