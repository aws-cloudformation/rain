package build

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var buildListFlag = false
var bareTemplate = false
var buildJSON = false
var promptFlag = false
var showSchema = false

// Borrowing a simplified SAM spec file from goformation
// Ideally we would autogenerate from the full SAM spec but that thing is huge
// Full spec: https://github.com/aws/serverless-application-model/blob/develop/samtranslator/schema/schema.json

//go:embed sam-2016-10-31.json
var samSpecSource string

func addScalar(n *yaml.Node, propName string, val string) error {
	if n.Kind == yaml.MappingNode {
		node.Add(n, propName, val)
	} else if n.Kind == yaml.SequenceNode {
		n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: val})
	} else {
		return fmt.Errorf("unexpected kind %v for %s:%s", n.Kind, propName, val)
	}
	return nil
}

// buildProp adds boilerplate code to the node, depending on the shape of the property
func buildProp(n *yaml.Node, propName string, prop cfn.Prop, schema cfn.Schema) error {

	config.Debugf("%s n: %s", propName, node.ToSJson(n))
	j, _ := json.Marshal(prop)
	config.Debugf("%s prop: %s", propName, j)

	switch prop.Type {
	case "string":
		if len(prop.Enum) > 0 {
			return addScalar(n, propName, strings.Join(prop.Enum, " or "))
		} else {
			return addScalar(n, propName, "STRING")
		}
	case "object":
		var objectProps *cfn.Prop
		if prop.Properties != nil {
			objectProps = &prop
		} else if len(prop.OneOf) > 0 {
			objectProps = prop.OneOf[0]
		} else if len(prop.AnyOf) > 0 {
			objectProps = prop.AnyOf[0]
		} else {
			config.Debugf("%s: object type without properties", propName)
			return addScalar(n, propName, "{JSON}")
		}
		if objectProps != nil {
			if n.Kind == yaml.MappingNode {
				// Make a mapping node and recurse to add sub properties
				m := node.AddMap(n, propName)
				return buildNode(m, objectProps, &schema)
			} else if n.Kind == yaml.SequenceNode {
				// We're adding objects to an array,
				// so we don't need the Scalar node for the name,
				// since propName will be a placeholder like 0 or 1
				j, _ := json.Marshal(objectProps)
				config.Debugf("array objectProps: %s", j)
				sequenceMap := &yaml.Node{Kind: yaml.MappingNode}
				n.Content = append(n.Content, sequenceMap)
				return buildNode(sequenceMap, objectProps, &schema)
			} else {
				return fmt.Errorf("unexpected kind %v for %s", n.Kind, propName)
			}
		}
	case "array":
		// Look at items to see what type is in the array
		if prop.Items != nil {
			// Add a sequence node, then add 2 sample elements
			sequenceName := &yaml.Node{Kind: yaml.ScalarNode, Value: propName}
			n.Content = append(n.Content, sequenceName)
			sequence := &yaml.Node{Kind: yaml.SequenceNode}
			n.Content = append(n.Content, sequence)
			var arrayItems *cfn.Prop

			// Resolve array items ref
			if prop.Items.Ref != "" {
				reffed := strings.Replace(prop.Items.Ref, "#/definitions/", "", 1)
				var hasDef bool
				if arrayItems, hasDef = schema.Definitions[reffed]; !hasDef {
					return fmt.Errorf("%s: Items.%s not found in definitions", propName, reffed)
				}
			} else {
				arrayItems = prop.Items
			}

			// Add the samples to the sequence node
			err := buildProp(sequence, "0", *arrayItems, schema)
			if err != nil {
				return err
			}
			err = buildProp(sequence, "1", *arrayItems, schema)
			if err != nil {
				return err
			}
			return nil

		} else {
			return fmt.Errorf("%s: array without items?", propName)
		}
	case "boolean":
		return addScalar(n, propName, "BOOLEAN")
	case "number":
		return addScalar(n, propName, "NUMBER")
	case "integer":
		return addScalar(n, propName, "INTEGER")
	case "":
		if prop.Ref != "" {
			reffed := strings.Replace(prop.Ref, "#/definitions/", "", 1)
			if objectProps, hasDef := schema.Definitions[reffed]; !hasDef {
				return fmt.Errorf("%s: blank type Ref %s not found in definitions", propName, reffed)
			} else {
				return buildProp(n, propName, *objectProps, schema)
			}
		} else {
			return fmt.Errorf("expected blank type to have $ref: %s", propName)
		}
	default:
		return fmt.Errorf("unexpected prop type for %s: %s", propName, prop.Type)
	}

	return nil
}

// buildNode recursively builds a node for a schema-like object
func buildNode(n *yaml.Node, s cfn.SchemaLike, schema *cfn.Schema) error {

	// Add all props or just the required ones
	if bareTemplate {
		for _, requiredName := range s.GetRequired() {
			if p, hasProp := schema.Properties[requiredName]; hasProp {
				err := buildProp(n, requiredName, *p, *schema)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("required: %s not found in properties", requiredName)
			}
		}
	} else {
		for k, p := range s.GetProperties() {
			err := buildProp(n, k, *p, *schema)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func startTemplate() cft.Template {

	t := cft.Template{}

	// Create the template header sections
	t.Node = &yaml.Node{Kind: yaml.DocumentNode, Content: make([]*yaml.Node, 0)}
	t.Node.Content = append(t.Node.Content,
		&yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0)})
	t.AddScalarSection(cft.AWSTemplateFormatVersion, "2010-09-09")
	t.AddScalarSection(cft.Description, "Generated by rain")

	return t
}

// isSAM returns true if the type is a SAM transform
func isSAM(typeName string) bool {
	transforms := []string{
		"AWS::Serverless::Function",
		"AWS::Serverless::Api",
		"AWS::Serverless::HttpApi",
		"AWS::Serverless::Application",
		"AWS::Serverless::SimpleTable",
		"AWS::Serverless::LayerVersion",
		"AWS::Serverless::StateMachine",
	}
	return slices.Contains(transforms, typeName)
}

func build(typeNames []string) (cft.Template, error) {

	t := startTemplate()

	// Add the Resources section
	resourceMap, err := t.AddMapSection(cft.Resources)
	if err != nil {
		return t, err
	}

	for _, typeName := range typeNames {

		var schema *cfn.Schema

		// Check to see if it's a SAM resource
		if isSAM(typeName) {
			t.AddScalarSection(cft.Transform, "AWS::Serverless-2016-10-31")

			// Convert the spec to a cfn.Schema and skip downloading from the registry
			schema, err = convertSAMSpec(samSpecSource, typeName)
			if err != nil {
				return t, err
			}

			j, _ := json.Marshal(schema)
			config.Debugf("Converted SAM schema: %s", j)

		} else {

			// Call CCAPI to get the schema for the resource
			schemaSource, err := cfn.GetTypeSchema(typeName)
			config.Debugf("schema source: %s", schemaSource)
			if err != nil {
				return t, err
			}

			// Parse the schema JSON string
			schema, err = cfn.ParseSchema(schemaSource)
			if err != nil {
				return t, err
			}
		}

		// Add a node for the resource
		shortName := strings.Split(typeName, "::")[2]
		r := node.AddMap(resourceMap, "My"+shortName)
		node.Add(r, "Type", typeName)
		props := node.AddMap(r, "Properties")

		// Recursively build the node
		err = buildNode(props, schema, schema)
		if err != nil {
			return t, err
		}

	}

	return t, nil
}

// Cmd is the build command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "build [<resource type>] or <prompt>",
	Short:                 "Create CloudFormation templates",
	Long:                  "Outputs a CloudFormation template containing the named resource types.",
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if buildListFlag {
			types, err := cfn.ListResourceTypes()
			if err != nil {
				panic(err)
			}
			for _, t := range types {
				fmt.Println(t)
			}
			return
		}

		if len(args) == 0 {
			cmd.Help()
			return
		}

		// --schema -s
		// Download and print out the registry schema
		if showSchema {
			schema, err := cfn.GetTypeSchema(args[0])
			if err != nil {
				panic(err)
			}
			fmt.Println(schema)
			return
		}

		// --prompt -p
		// Invoke Bedrock with Claude 2 to generate the template
		if promptFlag {
			prompt(strings.Join(args, " "))
			return
		}

		t, err := build(args)
		if err != nil {
			panic(err)
		}
		out := format.String(t, format.Options{
			JSON: buildJSON,
		})
		fmt.Println(out)
	},
}

func init() {
	Cmd.Flags().BoolVarP(&buildListFlag, "list", "l", false, "List all CloudFormation resource types")
	Cmd.Flags().BoolVarP(&promptFlag, "prompt", "p", false, "Generate a template using Bedrock and a prompt")
	Cmd.Flags().BoolVarP(&bareTemplate, "bare", "b", false, "Produce a minimal template, omitting all optional resource properties")
	Cmd.Flags().BoolVarP(&buildJSON, "json", "j", false, "Output the template as JSON (default format: YAML)")
	Cmd.Flags().BoolVarP(&showSchema, "schema", "s", false, "Output the registry schema for a resource type")
	Cmd.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
}
