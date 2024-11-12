package pkg

import (
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// omitIfs returns true if the resource should be omitted due to
// IfParam or IfNotParam, which are simplified Conditionals.
// IfParam means "if this parameter is set, show this resource"
// IfNotParam means "if this parameter is empty, show this resource"
func omitIfs(rainMetadata *yaml.Node, moduleParams *yaml.Node, templateProps *yaml.Node, moduleResource *yaml.Node) bool {

	retval := false

	ifp := s11n.GetValue(rainMetadata, IfParam)
	// If the value of IfParam is not set, omit this resource
	if ifp != "" {
		if moduleParams != nil &&
			s11n.GetMap(moduleParams, ifp) != nil &&
			s11n.GetValue(templateProps, ifp) == "" &&
			len(s11n.GetMap(templateProps, ifp)) == 0 {
			retval = true
		}
		// Get rid of the IfParam, since it's irrelevant in the resulting template
		node.RemoveFromMap(rainMetadata, IfParam)
	}

	ifnp := s11n.GetValue(rainMetadata, IfNotParam)
	// If the value of IfNotParam is set, omit this resource
	if ifnp != "" {
		moduleParamExists := s11n.GetMap(moduleParams, ifnp) != nil
		hasTemplatePropValue := s11n.GetValue(templateProps, ifnp) != ""
		hasTemplatePropMap := len(s11n.GetMap(templateProps, ifnp)) > 0
		existsInTemplateProps := hasTemplatePropValue || hasTemplatePropMap

		if moduleParamExists && existsInTemplateProps {
			retval = true
		}

		// Get rid of the IfParam, since it's irrelevant in the resulting template
		node.RemoveFromMap(rainMetadata, IfNotParam)
	}

	// If the Rain attribute is empty, get rid of it
	if len(rainMetadata.Content) == 0 {
		_, metadataNode, _ := s11n.GetMapValue(moduleResource, Metadata)
		node.RemoveFromMap(metadataNode, Rain)

		// Now if there is nothing left in Metadata, get rid of that too
		if len(metadataNode.Content) == 0 {
			node.RemoveFromMap(moduleResource, Metadata)
		}
	}

	return retval
}
