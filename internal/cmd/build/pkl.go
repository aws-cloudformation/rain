package build

import (
	"fmt"
	"slices"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
)

// Names that shouldn't be used as property names,
// but sometimes they are, so we suffix any property
// with one of these name with `Property`.
var reservedNames = []string{
	"Type",
	"Properties",
	"DependsOn",
	"CreationPolicy",
	"DeletionPolicy",
	"Metadata",
	"UpdatePolicy",
	"UpdateReplacePolicy",
}

type stringAlias struct {
	Name   string
	Values []string
}

// Represents a definition property
//
// for example
//
//	{
//	   Name: "ServerSideEncryptionConfiguration"
//	   Type: "Listing<ServerSideEncryptionConfiguration>"
//	}
type pklDefProp struct {
	Name string
	Type string
}

// A class that represents a definition in a registry resource schema
type pklDefClass struct {
	Name        string
	Description string
	Props       []*pklDefProp
	Aliases     []*stringAlias
}

// getDefName renames definition class names to avoid
// the duplicate class name problem with a resource has a
// definition with the exact same name as itself.
func getDefName(clsName string, name string) string {
	return clsName + name
}

// fixPropName appends Property to any reserved names
func fixPropName(propName string) string {
	if slices.Contains(reservedNames, propName) {
		return propName + "Property"
	}
	return propName
}

func printTypeAlias(alias *stringAlias) {
	fmt.Println()
	fmt.Printf("typealias %s = ", alias.Name)
	for i, v := range alias.Values {
		if i != 0 {
			fmt.Print("|")
		}
		fmt.Printf("\"%s\"", v)
	}
	fmt.Print("\n")
}

func printCls(cls *pklDefClass) {
	// Print the type aliases so they are above the relevant class

	for _, alias := range cls.Aliases {
		printTypeAlias(alias)
	}

	fmt.Println()

	printDescription(cls.Description, "")
	fmt.Printf("open class %s {\n", cls.Name)

	for _, prop := range cls.Props {
		fmt.Printf("    %s: %s\n", prop.Name, prop.Type)
	}

	fmt.Printf("}\n")
}

// Returns the alias name and adds it to the class
func createTypeAlias(defName string, propName string, cls *pklDefClass, enum []any) string {
	aliasName := fmt.Sprintf("%s%s", defName, propName)
	alias := &stringAlias{Name: aliasName, Values: make([]string, 0)}
	for _, e := range enum {
		alias.Values = append(alias.Values, fmt.Sprintf("%s", e))
	}
	cls.Aliases = append(cls.Aliases, alias)
	return aliasName
}

// getPropType gets the property type for a property or definition
func getPropType(defName string, propName string,
	prop *cfn.Prop, cls *pklDefClass, required bool, shortName string) (string, error) {

	prop.Type = cfn.ConvertPropType(prop.Type)

	var retval string
	switch prop.Type {
	case "string":
		if len(prop.Enum) > 0 {
			// Create a type alias
			// Example: typealias SSEAlgorithmTypes = "aws:kms"|"AES256"|"aws:kms:dsse"
			aliasName := createTypeAlias(defName, propName, cls, prop.Enum)
			retval = aliasName + "|Mapping"
		} else if len(prop.Pattern) > 0 {
			// BUG: Multiline regex
			// AWS::Omics::AnnotationStore
			// \n is getting converted to an actual newline
			//  "pattern": "^arn:([^:\n]*):([^:\n]*):([^:\n]*):([0-9]{12}):([^:\n]*)$"
			retval = fmt.Sprintf("String(matches(Regex(#\"%s\"#)))|Mapping", prop.Pattern)
		} else {
			retval = "String|Mapping"
		}
	case "object":
		retval = "Dynamic"
	case "array":
		if prop.Items != nil {
			if prop.Items.Ref != "" {
				clsName := getDefName(shortName, strings.Replace(prop.Items.Ref, "#/definitions/", "", 1))
				retval = fmt.Sprintf("Listing<%s>", clsName)
			} else {
				if prop.Items.Type == nil {
					prop.Items.Type = ""
				}
				// TODO: items that are objects like { oneOf [ ...
				// AWS::IoTFleetWise::DecoderManifest.SignalDecoders
				switch prop.Items.Type {
				case "string":
					if len(prop.Items.Enum) > 0 {
						aliasName := createTypeAlias(defName, propName, cls, prop.Items.Enum)
						retval = fmt.Sprintf("Listing<%s|Mapping>", aliasName)
					} else {
						retval = "Listing<String|Mapping>"
					}
				case "boolean":
					retval = "Listing<Boolean|Mapping>"
				case "number":
					retval = "Listing<Number|Mapping>"
				case "integer":
					retval = "Listing<Int|Mapping>"
				default:
					return "", fmt.Errorf("no item type for %s", propName)
				}
			}
		} else {
			return "", fmt.Errorf("array has no items: %s", propName)
		}
	case "boolean":
		retval = "Boolean|Mapping"
	case "number":
		retval = "Number|Mapping"
	case "integer":
		retval = "Int|Mapping"
	case "":
		if prop.Ref != "" {
			clsName := getDefName(shortName, strings.Replace(prop.Ref, "#/definitions/", "", 1))
			retval = clsName
		} else {
			return "", fmt.Errorf("expected blank type to have $ref: %s", propName)
		}
	}

	if retval == "" {
		return "", fmt.Errorf("unable to determine type for %s: %v", propName, prop.Type)
	}
	if !required {
		retval = fmt.Sprintf("(%s)?", retval)
	}
	return retval, nil
}

func printDescription(description string, indent string) {
	descTokens := strings.Split(description, "\n")
	for i, d := range descTokens {
		fmt.Printf("%s/// %s\n", indent, d)
		if i == 0 && len(descTokens) > 1 {
			fmt.Printf("%s///\n", indent)
		}
	}
}

func generatePklClass(typeName string) error {
	schema, err := getSchema(typeName)
	if err != nil {
		return err
	}

	shortName := strings.Split(typeName, "::")[2]

	fmt.Println("///", typeName)
	fmt.Println("///")
	fmt.Println("/// Generated by rain build --pkl-class", typeName)
	moduleName := strings.ToLower(strings.Replace(typeName, "::", ".", -1))
	// "function" can't be used in a module name
	moduleName = strings.Replace(moduleName, "function", "function_", -1)
	fmt.Println("module", moduleName)
	fmt.Println()

	fmt.Println("import \"../../cloudformation.pkl\"")

	classes := make([]*pklDefClass, 0)

	// Iterate over definitions, creating a class for each one
	for name, def := range schema.Definitions {
		cls := &pklDefClass{
			Name:        getDefName(shortName, name),
			Description: def.Description,
			Props:       make([]*pklDefProp, 0),
			Aliases:     make([]*stringAlias, 0),
		}
		classes = append(classes, cls)

		r := def.GetRequired()

		for propName, prop := range def.Properties {
			required := slices.Contains(r, propName)
			propType, err := getPropType(name, propName, prop, cls, required, shortName)
			if err != nil {
				return err
			}
			cls.Props = append(cls.Props, &pklDefProp{Name: propName, Type: propType})
		}
	}

	// Print out each of the classes
	for _, cls := range classes {
		printCls(cls)
	}

	// Create a class for the type itself
	fmt.Println()
	printDescription(schema.Description, "")
	fmt.Printf("open class %s extends cloudformation.Resource {\n", shortName)
	fmt.Println()
	fmt.Printf("    Type = \"%s\"\n", typeName)
	fmt.Println()

	propNames := make([]string, 0)
	cls := &pklDefClass{
		Name:        shortName,
		Description: schema.Description,
		Props:       make([]*pklDefProp, 0),
		Aliases:     make([]*stringAlias, 0),
	}
	requiredProps := schema.GetRequired()
	for propName, prop := range schema.Properties {
		propPath := "/properties/" + propName
		// Don't emit read-only properties
		if slices.Contains(schema.ReadOnlyProperties, propPath) {
			continue
		}
		required := slices.Contains(requiredProps, propName)
		propType, err := getPropType(shortName, propName, prop, cls, required, shortName)
		if err != nil {
			return err
		}
		fmt.Println()

		printDescription(prop.Description, "    ")
		fmt.Printf("    hidden %s: %s\n", fixPropName(propName), propType)
		propNames = append(propNames, propName)
	}

	fmt.Println()
	fmt.Printf("    Properties {\n")
	for _, propName := range propNames {
		fmt.Printf("        [\"%s\"] = if (%s == null) null else %s\n",
			propName, fixPropName(propName), fixPropName(propName))
	}
	fmt.Printf("    }\n\n")

	fmt.Println("}")

	if len(cls.Aliases) > 0 {
		fmt.Println()
		for _, alias := range cls.Aliases {
			printTypeAlias(alias)
		}
	}

	return nil
}
