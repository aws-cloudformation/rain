package build

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/config"
)

// #definition classes
var classes map[string]*pklDefClass

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

type aliasType int

const (
	MAP = iota
	LISTING
	STRINGS
	TYPES
	PRIMITIVE
)

type defAlias struct {
	Name           string
	Values         []string
	Type           aliasType
	PrimitiveValue string
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
	Aliases     []*defAlias

	// Extra classes needed by anyOfs that resulted in a new typealias
	OfClasses []*pklDefClass
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

// printTypeAlias prints out a type alias, which
// might be String, an enum, a listing, or a map
func printTypeAlias(alias *defAlias) {
	fmt.Println()
	fmt.Printf("typealias %s = ", alias.Name)

	switch alias.Type {
	case MAP:
		// map (patternProperties)
		fmt.Print("Mapping<String, Any>")
	case LISTING:
		fmt.Print("Listing<")
		// array
		for i, v := range alias.Values {
			if i != 0 {
				fmt.Print("|")
			}
			fmt.Printf("%s", v)
		}
		fmt.Print(">")
	case STRINGS:
		if len(alias.Values) > 0 {
			// enum
			for i, v := range alias.Values {
				if i != 0 {
					fmt.Print("|")
				}
				fmt.Printf("\"%s\"", v)
			}
		} else {
			fmt.Print("String")
		}
	case TYPES:
		for i, v := range alias.Values {
			if i != 0 {
				fmt.Print("|")
			}
			fmt.Printf("%s", v)
		}
	case PRIMITIVE:
		fmt.Print(alias.PrimitiveValue)
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

func createAlias(defName string, propName string, cls *pklDefClass, typeName string) string {
	aliasName := fmt.Sprintf("%s%s", defName, propName)
	alias := &defAlias{Name: aliasName, PrimitiveValue: typeName, Type: PRIMITIVE}
	cls.Aliases = append(cls.Aliases, alias)
	return aliasName
}

// Returns the alias name and adds it to the class
func createStringAlias(defName string, propName string, cls *pklDefClass, enum []any) string {
	aliasName := fmt.Sprintf("%s%s", defName, propName)
	alias := &defAlias{Name: aliasName, Values: make([]string, 0), Type: STRINGS}
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
			aliasName := createStringAlias(defName, propName, cls, prop.Enum)
			retval = aliasName + "|Mapping"
		} else if len(prop.Pattern) > 0 {
			// Multiline regex
			// AWS::Omics::AnnotationStore
			// \n was getting converted to an actual newline
			//  "pattern": "^arn:([^:\n]*):([^:\n]*):([^:\n]*):([0-9]{12}):([^:\n]*)$"
			retval = fmt.Sprintf("String(matches(Regex(#\"%s\"#)))|Mapping", prop.Pattern)
			retval = strings.Replace(retval, "\n", "\\n", -1)
			retval = strings.Replace(retval, "\r", "", -1)
		} else {
			retval = "String|Mapping"
		}
	case "object":
		if prop.PatternProperties != nil {
			// Create a type alias
			alias := &defAlias{Name: shortName + defName + propName, Type: MAP}
			cls.Aliases = append(cls.Aliases, alias)
			retval = shortName + defName + propName
		} else {
			retval = "Dynamic"
		}
	case "array":
		if prop.Items != nil {
			if prop.Items.Ref != "" {
				clsName := getDefName(shortName, strings.Replace(prop.Items.Ref, "#/definitions/", "", 1))
				retval = fmt.Sprintf("Listing<%s>", clsName)
			} else {
				if prop.Items.Type == nil {
					prop.Items.Type = ""
				}
				switch prop.Items.Type {
				case "string":
					if len(prop.Items.Enum) > 0 {
						aliasName := createStringAlias(defName, propName, cls, prop.Items.Enum)
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
				case "object":
					retval = "Listing<Dynamic>"
				default:
					if len(prop.Items.OneOf) > 0 {
						err := handleOfs(defName+propName, prop.Items.OneOf, shortName, cls)
						if err != nil {
							return "", err
						}
						retval = fmt.Sprintf("Listing<%s>", shortName+defName+propName)
					} else if len(prop.Items.AnyOf) > 0 {
						err := handleOfs(defName+propName, prop.Items.AnyOf, shortName, cls)
						if err != nil {
							return "", err
						}
						retval = fmt.Sprintf("Listing<%s>", shortName+defName+propName)
					} else {
						return "", fmt.Errorf("no item type for %s", propName)
					}
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
		} else if prop.PatternProperties != nil {
			// Create a type alias
			alias := &defAlias{Name: shortName + defName + propName, Type: MAP}
			cls.Aliases = append(cls.Aliases, alias)
			retval = shortName + defName + propName
		} else if len(prop.OneOf) > 0 {
			err := handleOfs(defName+propName, prop.OneOf, shortName, cls)
			if err != nil {
				return "", err
			}
			retval = shortName + defName + propName
		} else if len(prop.AnyOf) > 0 {
			err := handleOfs(defName+propName, prop.AnyOf, shortName, cls)
			if err != nil {
				return "", err
			}
			retval = shortName + defName + propName
		} else {
			return "", fmt.Errorf("expected blank type to have $ref, patternProperties, anyOf, or oneOf: %s", propName)
		}
	}

	//if retval == "" {
	//	return "", fmt.Errorf("unable to determine type for %s: %v", propName, prop.Type)
	//}

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

// handleOfs handles anyof, oneOf
func handleOfs(name string, of []*cfn.Prop, shortName string, defCls *pklDefClass) error {

	if len(of) == 0 {
		return fmt.Errorf("handleOfs length zero for %s", name)
	}

	// Make a new type alias for it
	aliasName := shortName + name
	alias := &defAlias{Name: aliasName, Values: make([]string, 0), Type: TYPES}
	for i, xOf := range of {
		// Create a new definition class for the type
		if xOf.Title == "" {
			xOf.Title = fmt.Sprintf("%d", i)
		}
		xOfCls, err := createDefinitionClass(name+xOf.Title, xOf, shortName)
		if err != nil {
			return fmt.Errorf("unable to create def class for xOf %s: %s: %v", name, xOf.Title, err)
		}
		if xOfCls.Name == "" {
			return fmt.Errorf("handleOfs empty class name for %s", name)
		}

		// TODO: What if the class has properties? We're not emitting it
		// We can't just add to classes, since we may have already printed them all
		// if we are now processing top level properties

		if defCls.OfClasses == nil {
			defCls.OfClasses = make([]*pklDefClass, 0)
		}
		defCls.OfClasses = append(defCls.OfClasses, xOfCls)

		alias.Values = append(alias.Values, xOfCls.Name)
	}
	defCls.Aliases = append(defCls.Aliases, alias)
	return nil
}

// createDefinitionClass creates classes based on #definitions and on oneOf types
func createDefinitionClass(name string, def *cfn.Prop, shortName string) (*pklDefClass, error) {
	cls := &pklDefClass{
		Name:        getDefName(shortName, name),
		Description: def.Description,
		Props:       make([]*pklDefProp, 0),
		Aliases:     make([]*defAlias, 0),
	}
	classes[name] = cls

	r := def.GetRequired()

	for propName, prop := range def.Properties {
		required := slices.Contains(r, propName)
		propType, err := getPropType(name, propName, prop, cls, required, shortName)
		if err != nil {
			return nil, err
		}
		cls.Props = append(cls.Props, &pklDefProp{Name: propName, Type: propType})
	}

	def.Type = cfn.ConvertPropType(def.Type)

	// patternProperties
	if def.PatternProperties != nil {
		// Create a type alias
		alias := &defAlias{Name: shortName + name, Type: MAP}
		cls.Aliases = append(cls.Aliases, alias)
	}

	if def.Type == "object" && len(cls.Props) == 0 && def.PatternProperties == nil {
		// Tags?
		alias := &defAlias{Name: shortName + name, Type: PRIMITIVE, PrimitiveValue: "Dynamic"}
		cls.Aliases = append(cls.Aliases, alias)
	}

	//if def.Type != "object" && def.Ref == "" && def.Type != nil {
	if def.Type != "object" && def.Ref == "" {
		if len(cls.Props) > 0 {
			//return nil, fmt.Errorf("unexpected: definition %s with type %s has %d props",
			//	name, def.Type, len(cls.Props))
			config.Debugf("unexpected: definition %s with type %s has %d props",
				name, def.Type, len(cls.Props))
		} else {

			if def.Type == nil {
				def.Type = ""
			}

			switch def.Type {
			case "array":
				aliasName := fmt.Sprintf("%s%s", shortName, name)
				alias := &defAlias{Name: aliasName, Values: make([]string, 0), Type: LISTING}
				propType, err := getPropType(name, "Array", def.Items, cls, false, shortName)
				if err != nil {
					return nil, fmt.Errorf("unable to create array alias for %s", name)
				}
				alias.Values = append(alias.Values, propType)
				cls.Aliases = append(cls.Aliases, alias)
			case "string":
				// Create a type definition instead
				createAlias(shortName+name, "", cls, "String|Mapping")
			case "integer":
				createAlias(shortName+name, "", cls, "Int|Mapping")
			case "number":
				createAlias(shortName+name, "", cls, "Number|Mapping")
			case "boolean":
				createAlias(shortName+name, "", cls, "Boolean|Mapping")
			default:
				if len(def.OneOf) > 0 {
					err := handleOfs(name, def.OneOf, shortName, cls)
					if err != nil {
						return nil, err
					}
				} else if len(def.AnyOf) > 0 {
					err := handleOfs(name, def.AnyOf, shortName, cls)
					if err != nil {
						return nil, err
					}
				}

				if len(def.AllOf) > 0 {
					return nil, fmt.Errorf("allOf unsupported: %s", name)
				}

				// Something else we missed?
				//return nil, fmt.Errorf("unable to create class for definition %s", name)
			}
		}
	}

	if len(cls.Props) == 0 && len(cls.Aliases) == 0 {

		// This might be an anyOf or oneOf that only defines $ref
		if def.Ref != "" {
			// Make a type alias
			defName := getDefName(shortName,
				strings.Replace(def.Ref, "#/definitions/", "", 1))
			createAlias(shortName+name, "", cls, defName)
		} else {
			return nil, fmt.Errorf("%s has no Props or Aliases: %+v", name, def)
		}

	}
	return cls, nil
}

func printClassOrAlias(name string, cls *pklDefClass) error {
	if len(cls.Props) > 0 {
		// This is an object definition
		printCls(cls)
	} else {
		// Print a type alias instead
		if len(cls.Aliases) == 0 {
			return fmt.Errorf(
				"unexpected: definition %s has no Props and no Aliases",
				name)
		}
		for _, alias := range cls.Aliases {
			printTypeAlias(alias)
		}
		fmt.Println()
	}
	return nil
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

	classes = make(map[string]*pklDefClass, 0)

	// Iterate over definitions, creating a class for each one
	// Sort the keys so we don't get a diff when we regenerate
	keys := make([]string, 0)
	for name := range schema.Definitions {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		def := schema.Definitions[name]
		_, err := createDefinitionClass(name, def, shortName)
		if err != nil {
			return err
		}
	}

	// Print out each of the classes
	for name, cls := range classes {
		if err := printClassOrAlias(name, cls); err != nil {
			return err
		}
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
		Aliases:     make([]*defAlias, 0),
	}
	requiredProps := schema.GetRequired()
	keys = make([]string, 0)
	for name := range schema.Properties {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, propName := range keys {
		prop := schema.Properties[propName]
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

	// Print out any extra classes we created for anyOf, oneOf
	if len(cls.OfClasses) > 0 {
		fmt.Println()
		for _, ofCls := range cls.OfClasses {
			if err := printClassOrAlias(ofCls.Name, ofCls); err != nil {
				return err
			}
		}
	}

	// Print out any type aliases we created for properties
	if len(cls.Aliases) > 0 {
		fmt.Println()
		for _, alias := range cls.Aliases {
			printTypeAlias(alias)
		}
	}

	return nil
}
