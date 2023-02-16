// This file implements !Rain::Module
package pkg

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// Convert the module into a node for the packaged template
func processModule(module *yaml.Node,
	outputNode *yaml.Node, t cft.Template,
	typeNode *yaml.Node, parent node.NodePair) (bool, error) {

	config.Debugf("processModule module: %v", module)

	p, _ := json.MarshalIndent(parent, "", "  ")
	config.Debugf("parent: %v", string(p))

	logicalName := parent.Key.Value
	config.Debugf("logicalName: %v", logicalName)

	// The parent arg is the map in the template resource's Content[1] that contains Type, Properties, etc

	// Make a new node that will hold our additions to the original template
	outputNode.Content = make([]*yaml.Node, 0)

	if module.Kind != yaml.DocumentNode {
		return false, errors.New("expected module to be a DocumentNode")
	}

	curNode := module.Content[0] // ScalarNode !!map

	// Locate the Resources: section in the module
	_, resources := s11n.GetMapValue(curNode, "Resources")

	if resources == nil {
		return false, errors.New("expected the module to have a Resources section")
	}

	// Locate the ModuleExtension: resource. There should be exactly 1.
	_, moduleExtension := s11n.GetMapValue(resources, "ModuleExtension")
	if moduleExtension == nil {
		return false, errors.New("expected the module to have a single ModuleExtension resource")
	}

	// Process the ModuleExtension resource.

	_, meta := s11n.GetMapValue(moduleExtension, "Metadata")
	if meta == nil {
		return false, errors.New("expected ModuleExtension.Metadata")
	}

	_, extends := s11n.GetMapValue(meta, "Extends")
	if extends == nil {
		return false, errors.New("expected ModuleExtension.Metadata.Extends")
	}

	// Create a new node to contain the extended resource.
	// We will replace the original node from the template.
	ext := &yaml.Node{}
	ext.Kind = yaml.MappingNode
	ext.Content = make([]*yaml.Node, 0)

	// Type:
	ext.Content = append(ext.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "Type"})
	ext.Content = append(ext.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: extends.Value})

	// Properties:
	ext.Content = append(ext.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "Properties"})
	props := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: make([]*yaml.Node, 0),
	}
	ext.Content = append(ext.Content, props)

	outputNode.Content = append(outputNode.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: logicalName, // This is the logical name of the resource from the parent template
	})
	outputNode.Content = append(outputNode.Content, ext)

	// Get additional resources and add them to the output
	for i, resource := range resources.Content {
		tag := resource.ShortTag()
		config.Debugf("resource: %v, %v", tag, resource.Value)
		if resource.Kind == yaml.MappingNode {
			name := resources.Content[i-1].Value
			if name != "ModuleExtension" {
				// This is an additional resource to be added
				config.Debugf("Adding additional resource to output node: %v", name)
				outputNode.Content = append(outputNode.Content, resources.Content[i-1])
				outputNode.Content = append(outputNode.Content, resource)
			}
		}
	}

	return true, nil
}

// Type: !Rain::Module
func module(n *yaml.Node, root string, t cft.Template, parent node.NodePair) (bool, error) {

	if len(n.Content) != 2 {
		return false, errors.New("expected !Rain::Module <URI>")
	}

	// TODO - remove these after testing
	config.Debugf("module node: %v", n)
	config.Debugf("module node.Content: %v", n.Content)
	for i, c := range n.Content {
		config.Debugf("module n.Content[%v]: %v %v", i, c.Kind, c.Value)
	}
	config.Debugf("module root: %v", root)

	uri := n.Content[1].Value
	// Is this a local file or a URL?
	if strings.HasPrefix(uri, "file://") {
		// Read the local file
		content, path, err := expectFile(n, root)
		if err != nil {
			return false, err
		}

		// Parse the file
		var node yaml.Node
		err = yaml.Unmarshal(content, &node)
		if err != nil {
			return false, err
		}

		// Transform
		parse.TransformNode(&node)
		_, err = transform(&node, filepath.Dir(path), t)
		if err != nil {
			return false, err
		}

		// Create a new node to represent the processed module
		var outputNode yaml.Node
		_, err = processModule(&node, &outputNode, t, n, parent)
		if err != nil {
			return false, err
		}

		j, _ := json.MarshalIndent(t.Node, "", "  ")
		config.Debugf("t: %v", string(j))

		// Find the resource node in the template
		_, resourceNode := s11n.GetMapValue(t.Node.Content[0], "Resources")
		if resourceNode == nil {
			return false, errors.New("expected template to have Resources")
		}

		// Insert into the template
		resourceNode.Content = append(resourceNode.Content, outputNode.Content...)
	} else if strings.HasPrefix(uri, "https://") {
		// Download the file and then parse it
		// TODO
	} else {
		return false, errors.New("expected either file://path or https://path")
	}

	return true, nil

}
