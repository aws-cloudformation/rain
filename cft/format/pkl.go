package format

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"gopkg.in/yaml.v3"
)

var PklPackageAlias string = "@cfn"

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
	sb.WriteString("{\n")
	for j := 0; j < len(n.Content); j++ {
		writeNode(sb, n.Content[j], indent+"    ")
	}
	w(sb, "%s}\n", indent)
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
		writeNode(sb, val, indent+"    ")
		if val.Kind != yaml.ScalarNode {
			w(sb, "%s}\n", indent)
		}
	}
	return nil
}

func writeNode(sb *strings.Builder, n *yaml.Node, indent string) error {
	switch n.Kind {
	case yaml.ScalarNode:
		w(sb, " \"%s\"\n", n.Value)
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

	for i := 0; i < len(m.Content); i += 2 {
		section := m.Content[i].Value
		val := m.Content[i+1]
		if err := addSection(cft.Section(section), val, &sb); err != nil {
			return "", fmt.Errorf("failed to add %s: %v", section, err)
		}
	}
	return sb.String(), nil
}
