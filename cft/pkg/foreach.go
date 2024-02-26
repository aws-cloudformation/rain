package pkg

import (
	"errors"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

type fnfe struct {
	fnForEachLogicalId string
	fnForEachSequence  *yaml.Node
}

// TODO: This was broken in the refactor, come back to it later
func handleForEach(
	moduleResources *yaml.Node,
	t cft.Template,
	logicalId string,
	outputNode *yaml.Node,
	moduleParams *yaml.Node,
	templateProps *yaml.Node) (*fnfe, error) {

	var fnForEachKey string
	var fnForEachName string
	var fnForEachItems *yaml.Node
	var fnForEach *yaml.Node
	var fnForEachSequence *yaml.Node
	var fnForEachLogicalId string

	// Fn::ForEach

	// Iterate through all resources and see if any of them start with Fn::ForEach
	for i := 0; i < len(moduleResources.Content); i += 2 {
		keyNode := moduleResources.Content[i]
		valueNode := moduleResources.Content[i+1]

		fnForEachKey = keyNode.Value

		if strings.HasPrefix(fnForEachKey, "Fn::ForEach") {
			//config.Debugf("Found a foreach: %v:\n%v", fnForEachKey, node.ToJson(valueNode))
			if valueNode.Kind != yaml.SequenceNode {
				return nil, errors.New("expected Fn::ForEach to be a sequence")
			}
			if len(valueNode.Content) != 3 {
				return nil, errors.New("expected Fn::ForEach to have 3 items")
			}
			// The Fn::ForEach intrinsic takes 3 array elements as input
			fnForEachName = valueNode.Content[0].Value
			fnForEachItems = node.Clone(valueNode.Content[1]) // TODO - Resolve refs
			feBody := valueNode.Content[2]

			// TODO: Items might be a Ref to a property set by the parent template
			// We need to try and resolve that Ref like any other

			if feBody.Kind != yaml.MappingNode {
				return nil, errors.New("expected Fn::ForEach Body to be a mapping")
			}

			fnForEachLogicalId = feBody.Content[0].Value
			//feOutputMap := feBody.Content[1]
			//config.Debugf("LogicalId: %v\nOutputMap: %v", fnForEachLogicalId, feOutputMap)

			// Store this for later as we handle special cases for moduleExtension
			fnForEach = valueNode
			//config.Debugf("Fn::ForEach fnForEach: %v", node.ToJson(fnForEach))

			// Create node that looks like a regular ModuleExtenstion resource
			//moduleExtension := node.Clone(feOutputMap)
			// TODO: There is no ModuleExtension now

			//config.Debugf("Fn::ForEach moduleExtension: %v", node.ToJson(moduleExtension))

			// Make sure the parent template has Transform: AWS::LanguageExtensions
			docMap := t.Node.Content[0]
			_, transformNode, _ := s11n.GetMapValue(docMap, "Transform")
			if transformNode == nil {
				//config.Debugf("Adding Transform node")
				docMap.Content = append(docMap.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "Transform"})
				docMap.Content = append(docMap.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "AWS::LanguageExtensions"})
			}
		}
	}

	if fnForEach != nil {
		// Add the key for the Fn::ForEach from the module

		// We need to alter the name of the key to make sure it's unique, if the
		// module is used more than once in a template.
		// In the module if we have Fn::ForEach::MakeHandles:
		// and in the parent template the logical id is ForEachTest
		// then the key will be Fn::ForEach::ForEachTestMakeHandles:
		fixedKey := strings.Replace(fnForEachKey, "Fn::ForEach::", "Fn::ForEach::"+logicalId, 1)

		outputNode.Content = append(outputNode.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fixedKey,
		})

		//config.Debugf("foreach items: %v", node.ToJson(fnForEachItems))
		//config.Debugf("foreach name: %v", fnForEachName)

		fnForEachSequence = &yaml.Node{}
		fnForEachSequence.Kind = yaml.SequenceNode
		fnForEachSequence.Content = make([]*yaml.Node, 0)
		outputNode.Content = append(outputNode.Content, fnForEachSequence)

		fnForEachSequence.Content = append(fnForEachSequence.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: fnForEachName})

		// The second array element is the list of items to iterate over. It might be
		// a Ref to a parameter that supplies the values. Resolve the Ref.
		var resolvedItems *yaml.Node
		if fnForEachItems.Kind == yaml.SequenceNode {
			// TODO - Do we need to resolve individual items themselves? Will this be handled elsewhere?
			resolvedItems = fnForEachItems
		} else if fnForEachItems.Kind == yaml.MappingNode {
			if fnForEachItems.Content[0].Value == "Ref" {
				refName := fnForEachItems.Content[1].Value
				//config.Debugf("Fn::ForEach resolving items Ref %v", refName)
				_, p, _ := s11n.GetMapValue(moduleParams, refName)
				if p != nil {
					// Look up the value provided in the template props
					_, refval, _ := s11n.GetMapValue(templateProps, refName)
					if refval != nil {
						// This should be a comma separated value that we need to convert to a sequence
						resolvedItems = ConvertCsvToSequence(refval.Value)
					} else {
						// If it's not there, do we have a default in the module params?
						_, d, _ := s11n.GetMapValue(p, "Default")
						if d != nil {
							resolvedItems = ConvertCsvToSequence(d.Value)
						} else {
							// If not, leave it alone
							resolvedItems = fnForEachItems
						}
					}
				} else {
					// This is not a Ref to a module parameter.
					// TODO - Can this be a ref to something else in the module?
					// Leave it alone
					resolvedItems = fnForEachItems
				}

			} else {
				return nil, errors.New("expected Fn::ForEach item map to be a Ref")
			}
		} else {
			return nil, errors.New("expected Fn::ForEach items to be a sequence or a map")
		}
		//config.Debugf("resolvedItems: %v", node.ToJson(resolvedItems))

		fnForEachSequence.Content = append(fnForEachSequence.Content, resolvedItems)

		//config.Debugf("fnForEachSequence: %v", node.ToJson(fnForEachSequence))
		return &fnfe{
			fnForEachLogicalId: fnForEachLogicalId,
			fnForEachSequence:  fnForEachSequence,
		}, nil
	}
	return nil, nil
}
