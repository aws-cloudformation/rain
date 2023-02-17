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

	// The parent arg is the map in the template resource's Content[1] that contains Type, Properties, etc
	p, _ := json.MarshalIndent(parent, "", "  ")
	config.Debugf("parent: %v", string(p))

	if parent.Key == nil {
		return false, errors.New("expected parent.Key to not be nil. The !Rain::Module directive should come after Type: ")
	}

	// Get the logical id of the resource we are transforming
	logicalId := parent.Key.Value
	config.Debugf("logicalId: %v", logicalId)

	_, templateProps := s11n.GetMapValue(parent.Value, "Properties") // Might be nil

	// Make a new node that will hold our additions to the original template
	outputNode.Content = make([]*yaml.Node, 0)

	if module.Kind != yaml.DocumentNode {
		return false, errors.New("expected module to be a DocumentNode")
	}

	curNode := module.Content[0] // ScalarNode !!map

	// Locate the Resources: section in the module
	_, moduleResources := s11n.GetMapValue(curNode, "Resources")

	if moduleResources == nil {
		return false, errors.New("expected the module to have a Resources section")
	}

	// Locate the ModuleExtension: resource. There should be exactly 1.
	_, moduleExtension := s11n.GetMapValue(moduleResources, "ModuleExtension")
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

	_, moduleProps := s11n.GetMapValue(moduleExtension, "Properties")
	if moduleProps == nil {
		return false, errors.New("expected ModuleExtension.Properties")
	}

	// Create a new node to contain the extended resource.
	// This will be added to the template, and the original resource node will be removed
	ext := &yaml.Node{}
	ext.Kind = yaml.MappingNode
	ext.Content = make([]*yaml.Node, 0)

	// Type:
	// Replace the !Rain::Module directive with the extended type from the module
	ext.Content = append(ext.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "Type"})
	ext.Content = append(ext.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: extends.Value})

	// Properties:
	ext.Content = append(ext.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "Properties"})

	// Start by cloning the properties in the module
	props := node.Clone(moduleProps)

	// Replace any property values overridden in the parent template
	if templateProps != nil {
		for i, tprop := range templateProps.Content {
			for j, mprop := range props.Content {
				// Property names are even-indexed array elements
				if tprop.Value == mprop.Value && i%2 == 0 && j%2 == 0 {
					config.Debugf("Found overridden prop %v", tprop.Value)
					// Is a clone good enough here? Could get weird.
					// Maybe we just require that you replace the entire property if it's nested
					// Otherwise we have to do a diff
					props.Content[j+1] = node.Clone(templateProps.Content[i+1])
				}
			}
		}
	} else {
		// This is Ok. It's not required to override any props.
		config.Debugf("templateProps is nil")
	}

	// TODO Metadata, etc

	ext.Content = append(ext.Content, props)

	// Add the extension to the output node
	outputNode.Content = append(outputNode.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: logicalId, // This is the logical name of the resource from the parent template
	})
	outputNode.Content = append(outputNode.Content, ext)

	// Get additional resources and add them to the output
	for i, resource := range moduleResources.Content {
		tag := resource.ShortTag()
		config.Debugf("resource: %v, %v", tag, resource.Value)
		if resource.Kind == yaml.MappingNode {
			name := moduleResources.Content[i-1].Value
			if name != "ModuleExtension" {
				// This is an additional resource to be added
				config.Debugf("Adding additional resource to output node: %v", name)
				outputNode.Content = append(outputNode.Content, moduleResources.Content[i-1])
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

	uri := n.Content[1].Value

	// Is this a local file or a URL?
	if strings.HasPrefix(uri, "file://") {
		// Read the local file
		content, path, err := expectFile(n, root)
		if err != nil {
			return false, err
		}

		// Parse the file
		var moduleNode yaml.Node
		err = yaml.Unmarshal(content, &moduleNode)
		if err != nil {
			return false, err
		}

		// Transform
		parse.TransformNode(&moduleNode)
		// TODO: I think this allows us to nest modules. Test it.
		_, err = transform(&moduleNode, filepath.Dir(path), t)
		if err != nil {
			return false, err
		}

		// Create a new node to represent the processed module
		var outputNode yaml.Node
		_, err = processModule(&moduleNode, &outputNode, t, n, parent)
		if err != nil {
			return false, err
		}

		// Find the resource node in the template
		_, resourceNode := s11n.GetMapValue(t.Node.Content[0], "Resources")
		if resourceNode == nil {
			return false, errors.New("expected template to have Resources")
		}

		j, _ := json.MarshalIndent(resourceNode, "", "  ")
		config.Debugf("resourceNode: %v", string(j))

		j, _ = json.MarshalIndent(outputNode, "", "  ")
		config.Debugf("outputNode: %v", string(j))

		// Remove the original from the template
		err = node.RemoveFromMap(resourceNode, parent.Key.Value)
		if err != nil {
			return false, err
		}

		// Insert the transformed resource into the template
		resourceNode.Content = append(resourceNode.Content, outputNode.Content...)

	} else if strings.HasPrefix(uri, "https://") {
		// Download the file and then parse it
		// TODO
	} else {
		return false, errors.New("expected either file://path or https://path")
	}

	return true, nil

}
