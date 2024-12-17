package build

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var buildListFlag = false
var bareTemplate = false
var buildJSON = false
var promptFlag = false
var showSchema = false
var omitPatches = false
var recommendFlag = false
var outFn = ""
var pklClass = false
var noCache = false
var onlyCache = false
var promptLanguage = "cfn"
var model string
var models map[string]string
var activeFormat string
var selectedFormat string
var checkIcon = "✅"

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

func fixRef(ref string) string {
	return strings.Replace(ref, "#/definitions/", "", 1)
}

// buildProp adds boilerplate code to the node, depending on the shape of the property
func buildProp(n *yaml.Node, propName string, prop cfn.Prop, schema cfn.Schema, ancestorTypes []string, bare bool) error {

	isCircular := false

	prop.Type = cfn.ConvertPropType(prop.Type)

	if prop.Type == "" && len(prop.OneOf) > 0 || len(prop.AnyOf) > 0 {
		prop.Type = "object"
	}

	if prop.Type == "" && prop.PatternProperties != nil {
		prop.Type = "object"
	}

	if prop.Type == "" && len(prop.Properties) > 0 {
		prop.Type = "object"
	}

	switch prop.Type {
	case "string":
		if len(prop.Enum) > 0 {
			sa := make([]string, 0)
			for _, s := range prop.Enum {
				sa = append(sa, fmt.Sprintf("%s", s))
			}
			return addScalar(n, propName, strings.Join(sa, " or "))
		} else if len(prop.Pattern) > 0 {
			return addScalar(n, propName, strings.Replace(prop.Pattern, "|", " or ", -1))
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
			return addScalar(n, propName, "{JSON}")
		}
		if objectProps != nil {
			// We don't need to check for cycles here, since
			// we will check when eventually buildProp is called again

			if n.Kind == yaml.MappingNode {
				// Make a mapping node and recurse to add sub properties
				m := node.AddMap(n, propName)
				return buildNode(m, objectProps, &schema, ancestorTypes, bare)
			} else if n.Kind == yaml.SequenceNode {
				// We're adding objects to an array,
				// so we don't need the Scalar node for the name,
				// since propName will be a placeholder like 0 or 1
				sequenceMap := &yaml.Node{Kind: yaml.MappingNode}
				n.Content = append(n.Content, sequenceMap)
				return buildNode(sequenceMap, objectProps, &schema, ancestorTypes, bare)
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
				reffed := fixRef(prop.Items.Ref)
				var hasDef bool
				if arrayItems, hasDef = schema.Definitions[reffed]; !hasDef {
					return fmt.Errorf("%s: Items.%s not found in definitions", propName, reffed)
				}

				// Whenever we see a Ref, we need to track it to avoid infinite recursion
				if slices.Contains(ancestorTypes, reffed) {
					isCircular = true
				}
				ancestorTypes = append(ancestorTypes, reffed)
			} else {
				arrayItems = prop.Items
			}

			// Stop infinite recursion when a prop refers to an ancestor
			if isCircular {
				return addScalar(sequence, "", "{CIRCULAR}")
			} else {

				// Add the samples to the sequence node
				err := buildProp(sequence, "0", *arrayItems, schema, ancestorTypes, bare)
				if err != nil {
					return err
				}
				err = buildProp(sequence, "1", *arrayItems, schema, ancestorTypes, bare)
				if err != nil {
					return err
				}
				return nil
			}

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
			reffed := fixRef(prop.Ref)
			if objectProps, hasDef := schema.Definitions[reffed]; !hasDef {
				return fmt.Errorf("%s: blank type Ref %s not found in definitions", propName, reffed)
			} else {
				// Whenever we see a Ref, we need to track it to avoid infinite recursion
				if slices.Contains(ancestorTypes, reffed) {
					isCircular = true
				}
				ancestorTypes = append(ancestorTypes, reffed)
				if isCircular {
					return addScalar(n, propName, "{CIRCULAR}")
				} else {
					return buildProp(n, propName, *objectProps, schema, ancestorTypes, bare)
				}
			}
		} else {
			config.Debugf("Missing Ref: %s, ancestors: %v, %+v",
				propName, ancestorTypes, prop)
			return fmt.Errorf("expected blank type to have $ref: %s", propName)
		}
	default:
		return fmt.Errorf("unexpected prop type for %s: %s", propName, prop.Type)
	}

	return nil
}

func sortKeys(m map[string]*cfn.Prop) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

// buildNode recursively builds a node for a schema-like object
func buildNode(n *yaml.Node, s cfn.SchemaLike, schema *cfn.Schema, ancestorTypes []string, bare bool) error {

	if bare {
		requiredProps := s.GetRequired()
		props := s.GetProperties()
		for _, requiredName := range requiredProps {
			p, hasProp := props[requiredName]
			if hasProp {
				err := buildProp(n, requiredName, *p, *schema, ancestorTypes, bare)
				if err != nil {
					return err
				}
			} else {
				config.Debugf("invalid: %+v", s)
				return fmt.Errorf("invalid schema: required property %s not found in properties", requiredName)
			}
		}
	} else {
		props := s.GetProperties()
		// Sort the properties so we get consistent output
		sortedKeys := sortKeys(props)
		for _, k := range sortedKeys {
			p := props[k]
			propPath := "/properties/"
			for _, ancestor := range ancestorTypes {
				propPath = ancestor + "/"
			}
			propPath += k
			// Don't emit read-only properties
			if slices.Contains(schema.ReadOnlyProperties, propPath) {
				continue
			}
			err := buildProp(n, k, *p, *schema, ancestorTypes, bare)
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

		if isSAM(typeName) {
			t.AddScalarSection(cft.Transform, "AWS::Serverless-2016-10-31")
		}

		var schema *cfn.Schema
		schema, err := getSchema(typeName)
		if err != nil {
			return t, err
		}

		// Add a node for the resource
		shortName := strings.Split(typeName, "::")[2]
		r := node.AddMap(resourceMap, shortName)
		node.Add(r, "Type", typeName)
		props := node.AddMap(r, "Properties")

		// Recursively build the node
		ancestorTypes := make([]string, 0)
		err = buildNode(props, schema, schema, ancestorTypes, bareTemplate)
		if err != nil {
			return t, err
		}

	}

	return t, nil
}

func output(out string) {
	if outFn != "" {
		os.WriteFile(outFn, []byte(out), 0644)
	} else {
		fmt.Println(out)
	}
}

func getCacheUsage() cfn.ResourceCacheUsage {
	cacheUsage := cfn.UseCacheNormally
	if noCache {
		cacheUsage = cfn.DoNotUseCache
	} else if onlyCache {
		cacheUsage = cfn.OnlyUseCache
	}
	return cacheUsage
}

func list(prefix string) {
	types, err := cfn.ListResourceTypes(getCacheUsage())
	if err != nil {
		panic(err)
	}
	for _, t := range types {
		show := false
		if prefix != "" {
			// Filter by a prefix
			if strings.HasPrefix(t, prefix) {
				show = true
			}
		} else {
			show = true
		}
		if show {
			output(t)
		}
	}
}

func schema(typeName string) {
	// Use the local converted SAM schemas for serverless resources
	if isSAM(typeName) {
		// Convert the spec to a cfn.Schema and skip downloading from the registry
		schema, err := convertSAMSpec(samSpecSource, typeName)
		if err != nil {
			panic(err)
		}

		j, _ := json.MarshalIndent(schema, "", "    ")
		output(string(j))
	} else {
		schema, err := cfn.GetTypeSchema(typeName, getCacheUsage())
		if err != nil {
			panic(err)
		}
		output(schema)
	}
}

func basicBuild(args []string) {
	t, err := build(args)
	if err != nil {
		panic(err)
	}
	out := format.String(t, format.Options{
		JSON: buildJSON,
	})
	output(out)
}

// Cmd is the build command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "build [<resource type>] or <prompt>",
	Short:                 "Create CloudFormation templates",
	Long:                  "The build command interacts with the CloudFormation registry to list types, output schema files, and build starter CloudFormation templates containing the named resource types.",
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {

		// --list -l
		// List resource types
		if buildListFlag {
			prefix := ""
			if len(args) > 0 {
				prefix = args[0]
			}
			list(prefix)
			return
		}

		// --recommend -r
		// Output a recommended architecture
		if recommendFlag {
			recommend(args)
			return
		}

		// --schema -s
		// Download and print out the registry schema
		if showSchema {
			if len(args) == 0 {
				panic("provide a resource type name")
			}
			schema(args[0])
			return
		}

		// --prompt -p
		// Invoke Bedrock with Claude 2 to generate the template
		if promptFlag {
			validLangs := []string{LANG_CFN, LANG_GUARD, LANG_REGO}
			if !slices.Contains(validLangs, promptLanguage) {
				panic(fmt.Sprintf("provide a valid --prompt-lang: %v", validLangs))
			}
			if len(args) == 0 {
				panic("provide a prompt")
			}
			runPrompt(strings.Join(args, " "))
			return
		}

		// --pkl-class
		// Generate a pkl class based on the schema
		if pklClass {
			if len(args) == 0 {
				panic("provide a type name")
			}
			typeName := args[0]
			if err := generatePklClass(typeName); err != nil {
				panic(err)
			}
			return
		}

		if len(args) == 0 {
			// Enter interactive mode if we got this far with no args
			interactive()
		} else {
			// Basic build functionality
			basicBuild(args)
		}
	},
}

const (
	LANG_CFN   = "cfn"
	LANG_GUARD = "guard"
	LANG_REGO  = "rego"
)

func init() {
	models = make(map[string]string)
	models["claude2"] = "anthropic.claude-v2:1"
	models["claude3opus"] = "anthropic.claude-3-opus-20240229-v1:0"
	models["claude3sonnet"] = "anthropic.claude-3-sonnet-20240229-v1:0"
	models["claude3haiku"] = "anthropic.claude-3-haiku-20240307-v1:0"
	models["claude3.5sonnet"] = "anthropic.claude-3-5-sonnet-20240620-v1:0"

	activeFormat = " {{ .Name | magenta }}: {{ .Text | magenta }}"
	selectedFormat = " {{ .Name | magenta }}: {{ .Text | blue }}"

	if console.NoColour {
		activeFormat = " {{ .Name }}: {{ .Text }}"
		selectedFormat = " {{ .Name }}: {{ .Text }}"
	}

	Cmd.Flags().BoolVarP(&buildListFlag, "list", "l", false, "List all CloudFormation resource types with an optional name prefix")
	Cmd.Flags().BoolVar(&promptFlag, "prompt", false, "Generate a template using Bedrock and a prompt")
	Cmd.Flags().BoolVarP(&bareTemplate, "bare", "b", false, "Produce a minimal template, omitting all optional resource properties")
	Cmd.Flags().BoolVarP(&buildJSON, "json", "j", false, "Output the template as JSON (default format: YAML)")
	Cmd.Flags().BoolVarP(&showSchema, "schema", "s", false, "Output the raw un-patched registry schema for a resource type")
	Cmd.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
	Cmd.Flags().BoolVar(&omitPatches, "omit-patches", false, "Omit patches and use the raw schema")
	Cmd.Flags().BoolVar(&recommendFlag, "recommend", false, "Output a recommended architecture for the chosen use case")
	Cmd.Flags().StringVarP(&outFn, "output", "o", "", "Output to a file")
	Cmd.Flags().BoolVar(&pklClass, "pkl-class", false, "Output a pkl class based on a resource type schema")
	Cmd.Flags().BoolVar(&noCache, "no-cache", false, "Do not used cached schema files")
	Cmd.Flags().BoolVar(&noCache, "only-cache", false, "Only use cached schema files")
	Cmd.Flags().StringVar(&promptLanguage, "prompt-lang", "cfn", "The language to target for --prompt, CloudFormation YAML (cfn), CloudFormation Guard (guard), Open Policy Agent Rego (rego)")
	Cmd.Flags().StringVar(&model, "model", "claude2", "The ID of the Bedrock model to use for --prompt. Shorthand: claude2, claude3haiku, claude3sonnet, claude3opus, claude3.5sonnet")
}
