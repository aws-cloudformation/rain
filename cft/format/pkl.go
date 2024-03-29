package format

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"gopkg.in/yaml.v3"
)

var PklPackageAlias string = "@cfn"

func getClassName(resource *yaml.Node) string {
	typeName := getTypeName(resource)
	tokens := strings.Split(typeName, "::")
	if len(tokens) != 3 {
		return ""
	}
	return tokens[2]
}

func getTypeName(resource *yaml.Node) string {
	var typeName string
	for i := 0; i < len(resource.Content); i += 2 {
		attrName := resource.Content[i].Value
		attr := resource.Content[i+1]

		if attrName == "Type" {
			typeName = attr.Value
			break
		}
	}
	return typeName
}

// getModulePath gets the module name from the resource type.
// Returns "" if we don't have a class for that resource
func getModulePath(resource *yaml.Node) string {

	typeName := getTypeName(resource)

	if !strings.HasPrefix(typeName, "AWS") {
		return ""
	}

	typeName = strings.ToLower(typeName)

	// AWS::S3::Bucket = aws/s3/bucket.pkl
	return strings.Replace(typeName, "::", "/", -1) + ".pkl"
}

// getModuleNameFromPath gets the short module name
// aws/s3/bucket.pkl -> bucket
func getModuleNameFromPath(path string) string {
	path = strings.Replace(path, ".pkl", "", 1)
	tokens := strings.Split(path, "/")
	if len(tokens) != 3 {
		return ""
	}
	return tokens[2]
}

func writeResource(sb *strings.Builder, name string, resource *yaml.Node) error {
	indent := "    "
	modulePath := getModulePath(resource)
	moduleName := getModuleNameFromPath(modulePath)
	className := getClassName(resource)

	// If modulePath and moduleName are blank, we need to print out
	// the resource in the verbose way, without using classes.
	// This is likely a 3p resource that does not appear in the registry

	// TODO: Make "raw" an arg so we can output CloudFormation YAML with no imports

	if moduleName == "" {
		w(sb, "%s[\"%s\"] {\n", indent, name)
	} else {
		w(sb, "%s[\"%s\"] = new %s.%s {\n", indent, name, moduleName, className)
	}
	for i := 0; i < len(resource.Content); i += 2 {
		attrName := resource.Content[i].Value
		attr := resource.Content[i+1]
		if attrName == "Properties" && moduleName != "" {
			// In our generated classes we push all properties up to the top level
			// for convenience and type safety. (There are a few clashes we have to handle)
			for j := 0; j < len(attr.Content); j += 2 {
				propName := attr.Content[j].Value
				prop := attr.Content[j+1]
				w(sb, "%s%s", indent+"    ", propName)
				switch prop.Kind {
				case yaml.ScalarNode:
					sb.WriteString(" = ")
					writeNode(sb, prop, "")
				case yaml.SequenceNode:
					fallthrough
				case yaml.MappingNode:
					// TODO: Need to modify the writes so we use classes instead of maps
					// Are going to need reflection for this? Or can we predict class names?
					sb.WriteString(" = new Mapping {\n")
					writeNode(sb, prop, indent+"        ")
					sb.WriteString(indent + "    }\n")
				}
			}

		} else {
			w(sb, "    %s%s", indent, attrName)
			switch attr.Kind {
			case yaml.ScalarNode:
				sb.WriteString(" = ")
				writeNode(sb, attr, "")
			case yaml.SequenceNode:
				fallthrough
			case yaml.MappingNode:
				sb.WriteString(" {\n")
				writeNode(sb, attr, indent+"        ")
				sb.WriteString(indent + "    }\n")
			}
		}
	}

	sb.WriteString(indent + "}\n\n")
	return nil
}

func writeParameter(sb *strings.Builder, name string, param *yaml.Node) error {
	w(sb, "    [\"%s\"] {\n", name)
	for j := 0; j < len(param.Content); j += 2 {
		paramAttribute := param.Content[j].Value
		paramVal := param.Content[j+1]
		switch paramAttribute {
		case "AllowedValues":
			if paramVal.Kind != yaml.SequenceNode {
				config.Debugf("AllowedValues: %s", node.ToSJson(paramVal))
				return fmt.Errorf("expected Parameter %s AllowedValues to be a SequenceNode", name)
			}
			w(sb, "        %s {\n", paramAttribute)
			for k := 0; k < len(paramVal.Content); k++ {
				w(sb, "            \"%s\"\n", paramVal.Content[k].Value)
			}
			sb.WriteString("        }\n")
		case "Name":
			fallthrough
		case "Default":
			fallthrough
		case "Type":
			fallthrough
		case "Description":
			w(sb, "        %s = \"%s\"\n", paramAttribute, paramVal.Value)
		case "MinLength":
			fallthrough
		case "MaxLength":
			fallthrough
		case "MinValue":
			fallthrough
		case "MaxValue":
			fallthrough
		case "NoEcho":
			w(sb, "        %s = %s\n", paramAttribute, paramVal.Value)
		}

	}

	sb.WriteString("    }\n")
	return nil
}

func w(sb *strings.Builder, f string, args ...any) {
	sb.WriteString(fmt.Sprintf(f, args...))
}

// writeSequence writes a generic sequence
func writeSequence(sb *strings.Builder, n *yaml.Node, indent string) error {
	for i := 0; i < len(n.Content); i++ {
		switch n.Content[i].Kind {
		case yaml.ScalarNode:
			writeNode(sb, n.Content[i], indent)
		case yaml.MappingNode:
			sb.WriteString(indent + " new {\n")
			writeNode(sb, n.Content[i], indent+"    ")
			sb.WriteString(indent + "}\n")
		case yaml.SequenceNode:
			writeNode(sb, n.Content[i], indent)
		}
	}
	return nil
}

// writeMap writes out a generic Mapping
func writeMap(sb *strings.Builder, n *yaml.Node, indent string) error {
	if n.Kind != yaml.MappingNode {
		return errors.New("expected Mappings to be a MappingNode")
	}
	if len(n.Content)%2 != 0 {
		return errors.New("expected Content length to be even")
	}
	for i := 0; i < len(n.Content); i += 2 {
		name := n.Content[i].Value
		val := n.Content[i+1]
		w(sb, "%s[\"%s\"]", indent, name)
		if val.Kind == yaml.ScalarNode {
			sb.WriteString(" = ")
		} else {
			sb.WriteString(" {\n")
		}
		if val.Kind == yaml.ScalarNode {
			writeNode(sb, val, "")
		} else {
			writeNode(sb, val, indent+"    ")
			w(sb, "%s}\n", indent)
		}
	}
	// TODO: Rewrite intrinsic functions to use cfn. helpers
	return nil
}

func writeNode(sb *strings.Builder, n *yaml.Node, indent string) error {
	switch n.Kind {
	case yaml.ScalarNode:
		w(sb, "%s\"%s\"\n", indent, n.Value)
	case yaml.SequenceNode:
		return writeSequence(sb, n, indent)
	case yaml.MappingNode:
		return writeMap(sb, n, indent)
	}
	return nil
}

func addSection(section cft.Section, n *yaml.Node, sb *strings.Builder) error {
	switch section {
	case cft.AWSTemplateFormatVersion:
		fallthrough
	case cft.Description:
		w(sb, "%s = \"%s\"\n", section, n.Value)
	case cft.Parameters:
		if n.Kind != yaml.MappingNode {
			return errors.New("expected Parameters to be a MappingNode")
		}
		w(sb, "%s {\n", section)
		for i := 0; i < len(n.Content); i += 2 {
			writeParameter(sb, n.Content[i].Value, n.Content[i+1])
		}
		sb.WriteString("}\n")
	case cft.Resources:
		if n.Kind != yaml.MappingNode {
			return errors.New("expected Resources to be a MappingNode")
		}
		w(sb, "%s {\n", section)
		for i := 0; i < len(n.Content); i += 2 {
			writeResource(sb, n.Content[i].Value, n.Content[i+1])
		}
		sb.WriteString("}\n")
	case cft.Mappings:
		fallthrough
	case cft.Metadata:
		w(sb, "%s {\n", section)
		if err := writeMap(sb, n, "    "); err != nil {
			return fmt.Errorf("unable to write %s section: %v", section, err)
		}
		sb.WriteString("}\n")
	case cft.Rules:
	case cft.Conditions:
		w(sb, "%s {\n", section)
		if err := writeMap(sb, n, "    "); err != nil {
			return fmt.Errorf("unable to write %s section: %v", section, err)
		}
		sb.WriteString("}\n")
	case cft.Transform:
	case cft.Outputs:
	}

	return nil
}

// CftToPkl serializes the template as pkl.
// It assumes that the user is import the cloudformation package
func CftToPkl(t cft.Template) (string, error) {
	if t.Node == nil || len(t.Node.Content) != 1 {
		return "", errors.New("expected t.Node.Content[0]")
	}
	m := t.Node.Content[0]
	if len(m.Content)%2 != 0 {
		return "", errors.New("expected even number of map elements")
	}
	var sb strings.Builder
	w(&sb, "amends \"%s/template.pkl\"\n", PklPackageAlias)
	w(&sb, "import \"%s/cloudformation.pkl\" as cfn\n", PklPackageAlias)

	// Peek at all resources and import their types
	resources, err := t.GetSection(cft.Resources)
	if err != nil {
		return "", err
	}
	imports := make([]string, 0)
	for i := 0; i < len(resources.Content); i += 2 {
		resource := resources.Content[i+1]
		modulePath := getModulePath(resource)
		if modulePath != "" {
			if !slices.Contains(imports, modulePath) {
				imports = append(imports, modulePath)
				w(&sb, "import \"%s/%s\"\n", PklPackageAlias, modulePath)
			}
		}
	}

	// Write each section
	for i := 0; i < len(m.Content); i += 2 {
		section := m.Content[i].Value
		val := m.Content[i+1]
		if err := addSection(cft.Section(section), val, &sb); err != nil {
			return "", fmt.Errorf("failed to add %s: %v", section, err)
		}
	}
	return sb.String(), nil
}
