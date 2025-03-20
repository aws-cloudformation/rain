package format

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

const (
	SUB       = "Fn::Sub"
	REF       = "Ref"
	GETATT    = "Fn::GetAtt"
	EQUALS    = "Fn::Equals"
	CONTAINS  = "Fn::Contains"
	FINDINMAP = "Fn::FindInMap"
	GETAZS    = "Fn::GetAZs"
	SELECT    = "Fn::Select"
	SPLIT     = "Fn::Split"
)

func isIntrinsic(name string) bool {
	intrinsics := []string{
		SUB,
		REF,
		GETATT,
		EQUALS,
		CONTAINS,
		FINDINMAP,
		GETAZS,
		SELECT,
		SPLIT,
	}
	return slices.Contains(intrinsics, name)
}

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

func writeResource(sb *strings.Builder, name string, resource *yaml.Node, basic bool) error {
	indent := "    "
	modulePath := getModulePath(resource)
	moduleName := getModuleNameFromPath(modulePath)
	className := getClassName(resource)

	// If modulePath and moduleName are blank, we need to print out
	// the resource in the verbose way, without using classes.
	// This is likely a 3p resource that does not appear in the registry

	if moduleName == "" {
		basic = true
	}

	// ["LogicalId"] =
	if basic {
		w(sb, "%s[\"%s\"] {\n", indent, name)
	} else {
		w(sb, "%s[\"%s\"] = new %s.%s {\n", indent, name, moduleName, className)
	}

	for i := 0; i < len(resource.Content); i += 2 {
		attrName := resource.Content[i].Value
		attr := resource.Content[i+1]
		if attrName == "Properties" && !basic {
			// In our generated classes we push all properties up to the top level
			// for convenience and type safety. (There are a few clashes we have to handle)
			for j := 0; j < len(attr.Content); j += 2 {
				propName := attr.Content[j].Value
				prop := attr.Content[j+1]
				w(sb, "%s%s", indent+"    ", propName)
				switch prop.Kind {
				case yaml.ScalarNode:
					sb.WriteString(" = ")
					writeNode(sb, prop, "", basic)
				case yaml.SequenceNode:
					fallthrough
				case yaml.MappingNode:
					if isIntrinsic(prop.Content[0].Value) {
						w(sb, " = ")
						writeNode(sb, prop, indent+"        ", basic)
						w(sb, "\n")
					} else {
						w(sb, " {\n")
						writeNode(sb, prop, indent+"        ", basic)
						sb.WriteString(indent + "    }\n")
					}
				}
			}

		} else {
			if attrName == "Type" && !basic {
				// Don't bother outputting type, since it's in the module
				continue
			}
			if basic {
				// Basic rendering, without using module classes
				w(sb, "    %s[\"%s\"]", indent, attrName)
			} else {
				w(sb, "    %s%s ", indent, attrName)
			}
			switch attr.Kind {
			case yaml.ScalarNode:
				sb.WriteString(" = ")
				writeNode(sb, attr, "", basic)
			case yaml.SequenceNode:
				fallthrough
			case yaml.MappingNode:
				sb.WriteString(" {\n")
				writeNode(sb, attr, indent+"        ", basic)
				sb.WriteString(indent + "    }\n")
			}
		}
	}

	sb.WriteString(indent + "}\n\n")
	return nil
}

func writeOutput(sb *strings.Builder, name string, output *yaml.Node, basic bool) error {
	w(sb, "    [\"%s\"] = new cfn.Output {\n", name)

	for i := 0; i < len(output.Content); i += 2 {

		/*
			/// A stack Output exported value
			open class Export {
				Name: RefString
			}

			/// A stack output value
			open class Output {
				Description: RefString?
				Value: RefString
				Export: Export?
			}
		*/

		attrName := output.Content[i].Value
		attrValue := output.Content[i+1]

		switch attrName {
		case "Description":
			w(sb, "        Description = %s\n", attrValue.Value)
		case "Value":
			if attrValue.Kind == yaml.MappingNode {
				w(sb, "        Value = ")
				writeMap(sb, attrValue, "    ", basic)
			} else {
				w(sb, "        Value = %s\n", attrValue.Value)
			}
		case "Export":
			w(sb, "        Export = new cfn.Export {\n")
			_, nameNode, _ := s11n.GetMapValue(attrValue, "Name")
			if nameNode == nil {
				return errors.New("expected Export to have Name")
			}
			exportName := nameNode.Value
			w(sb, "            Name = %s\n", exportName)
			w(sb, "        }\n")
		}

	}

	w(sb, "    }\n")

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
func writeSequence(sb *strings.Builder, n *yaml.Node, indent string, basic bool) error {
	for i := 0; i < len(n.Content); i++ {
		item := n.Content[i]
		switch item.Kind {
		case yaml.ScalarNode:
			writeNode(sb, item, indent, basic)
		case yaml.MappingNode:
			if !basic && isIntrinsic(item.Content[0].Value) {
				w(sb, indent)
				writeNode(sb, item, indent+"        ", basic)
			} else {
				sb.WriteString(indent + " new {\n")
				writeNode(sb, item, indent+"    ", basic)
				sb.WriteString(indent + "}\n")
			}
		case yaml.SequenceNode:
			writeNode(sb, item, indent, basic)
		}
	}
	return nil
}

// writeMap writes out a generic Mapping
func writeMap(sb *strings.Builder, n *yaml.Node, indent string, basic bool) error {
	if n.Kind != yaml.MappingNode {
		return errors.New("expected Mappings to be a MappingNode")
	}
	if len(n.Content)%2 != 0 {
		return errors.New("expected Content length to be even")
	}
	for i := 0; i < len(n.Content); i += 2 {
		name := n.Content[i].Value
		val := n.Content[i+1]
		hasSpecialChars := false
		if strings.Contains(name, ":") || strings.Contains(name, "/") {
			hasSpecialChars = true
		}
		if isIntrinsic(name) {
			hasSpecialChars = false
		}
		if basic || hasSpecialChars {
			w(sb, "%s[\"%s\"]", indent, name)
			if val.Kind == yaml.ScalarNode {
				sb.WriteString(" = ")
				writeNode(sb, val, "", basic)
			} else {
				sb.WriteString(" {\n")
				writeNode(sb, val, indent+"    ", basic)
				w(sb, "%s}\n", indent)
			}
		} else {
			switch name {
			case SUB:
				w(sb, "cfn.Sub(\"%s\")\n", val.Value)
			case REF:
				w(sb, "cfn.Ref(\"%s\")\n", val.Value)
			case GETATT:
				w(sb, "cfn.GetAtt(\"%s\", \"%s\")\n",
					val.Content[0].Value, val.Content[1].Value)
			case EQUALS:
				w(sb, "cfn.Equals(\"%s\", \"%s\")\n",
					val.Content[0].Value, val.Content[1].Value)
			case CONTAINS:
				w(sb, "cfn.Contains(\"%s\", \"%s\")\n",
					val.Content[0].Value, val.Content[1].Value)
			case FINDINMAP:
				w(sb, "cfn.FindInMap(\"%s\", \"%s\", \"%s\")\n",
					val.Content[0].Value, val.Content[1].Value, val.Content[2].Value)
			case GETAZS:
				w(sb, "cfn.GetAZs(\"%s\")\n", val.Value)
			case SELECT:
				w(sb, "cfn.Select(\"%s\", ", val.Content[0].Value)
				writeNode(sb, val.Content[1], indent, basic)
				w(sb, "%s)\n", indent)
			case SPLIT:
				w(sb, "cfn.Split(\"%s\", ", val.Content[0].Value)
				writeNode(sb, val.Content[1], indent, basic)
				w(sb, "%s)\n", indent)
			default:
				w(sb, "%s%s", indent, name)
				if val.Kind == yaml.ScalarNode {
					sb.WriteString(" = ")
					writeNode(sb, val, "", basic)
				} else {
					if len(val.Content) > 0 && isIntrinsic(val.Content[0].Value) {
						w(sb, " = ")
						writeNode(sb, val, indent+"        ", basic)
					} else {
						w(sb, " {\n")
						writeNode(sb, val, indent+"        ", basic)
						sb.WriteString(indent + "    }\n")
					}
				}
			}
		}
	}
	return nil
}

func isNum(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func fixScalar(s string) string {
	// This is hard to do correctly without inspecting the schema.
	// Sometimes booleans and numbers are strings and YAML doesn't care
	// Also booleans are extra complicated

	boolStrings := []string{
		"true",
		"false",
	}

	if slices.Contains(boolStrings, s) {
		return s
	}

	if isNum(s) {
		return s
	}

	return fmt.Sprintf("\"%s\"", strings.Replace(s, "\n", "\\n", -1))
}

func writeNode(sb *strings.Builder, n *yaml.Node, indent string, basic bool) error {
	switch n.Kind {
	case yaml.ScalarNode:
		w(sb, "%s%s\n", indent, fixScalar(n.Value))
	case yaml.SequenceNode:
		return writeSequence(sb, n, indent, basic)
	case yaml.MappingNode:
		return writeMap(sb, n, indent, basic)
	}
	return nil
}

func addSection(section cft.Section, n *yaml.Node, sb *strings.Builder, basic bool) error {
	switch section {
	case cft.AWSTemplateFormatVersion:
		fallthrough
	case cft.Description:
		w(sb, "%s = %s\n", section, fixScalar(n.Value))
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
			writeResource(sb, n.Content[i].Value, n.Content[i+1], basic)
		}
		sb.WriteString("}\n")
	case cft.Mappings:
		fallthrough
	case cft.Metadata:
		w(sb, "%s {\n", section)
		if err := writeMap(sb, n, "    ", basic); err != nil {
			return fmt.Errorf("unable to write %s section: %v", section, err)
		}
		sb.WriteString("}\n")
	case cft.Rules:
	case cft.Conditions:
		w(sb, "%s {\n", section)
		if err := writeMap(sb, n, "    ", basic); err != nil {
			return fmt.Errorf("unable to write %s section: %v", section, err)
		}
		sb.WriteString("}\n")
	case cft.Transform:
	case cft.Outputs:
		w(sb, "%s {\n", section)
		for i := 0; i < len(n.Content); i += 2 {
			writeOutput(sb, n.Content[i].Value, n.Content[i+1], basic)
		}
		sb.WriteString("}\n")
	}

	return nil
}

// CftToPkl serializes the template as pkl.
// It assumes that the user is importing the cloudformation package
func CftToPkl(t *cft.Template, basic bool, pklPackageAlias string) (string, error) {
	if t.Node == nil || len(t.Node.Content) != 1 {
		return "", errors.New("expected t.Node.Content[0]")
	}
	m := t.Node.Content[0]
	if len(m.Content)%2 != 0 {
		return "", errors.New("expected even number of map elements")
	}
	var sb strings.Builder

	if !basic {
		w(&sb, "amends \"%s/template.pkl\"\n", pklPackageAlias)
		w(&sb, "import \"%s/cloudformation.pkl\" as cfn\n", pklPackageAlias)

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
					w(&sb, "import \"%s/%s\"\n", pklPackageAlias, modulePath)
				}
			}
		}
	}

	// Write each section
	for i := 0; i < len(m.Content); i += 2 {
		sb.WriteString("\n")
		section := m.Content[i].Value
		val := m.Content[i+1]
		if err := addSection(cft.Section(section), val, &sb, basic); err != nil {
			return "", fmt.Errorf("failed to add %s: %v", section, err)
		}
	}
	return sb.String(), nil
}
