package build

import (
	"fmt"
	"slices"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
)

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
	Name    string
	Props   []*pklDefProp
	Aliases []*stringAlias
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

	fmt.Printf("open class %s {\n", cls.Name)

	for _, prop := range cls.Props {
		fmt.Printf("    %s: %s\n", prop.Name, prop.Type)
	}

	fmt.Printf("}\n")
}

func getPropType(defName string, propName string,
	prop *cfn.Prop, cls *pklDefClass, required bool) (string, error) {

	var retval string
	switch prop.Type {
	case "string":
		if len(prop.Enum) > 0 {
			// Create a type alias
			// Example: typealias SSEAlgorithmTypes = "aws:kms"|"AES256"|"aws:kms:dsse"
			aliasName := fmt.Sprintf("%s%s", defName, propName)
			alias := &stringAlias{Name: aliasName, Values: make([]string, 0)}
			for _, e := range prop.Enum {
				alias.Values = append(alias.Values, fmt.Sprintf("%s", e))
			}
			cls.Aliases = append(cls.Aliases, alias)
			retval = aliasName + "|Mapping"
		} else if len(prop.Pattern) > 0 {
			retval = fmt.Sprintf("String(matches(Regex(%s)))|Mapping", prop.Pattern)
		} else {
			retval = "String|Mapping"
		}
	case "object":
		// TODO
	case "array":
		// TODO
	case "boolean":
		// TODO
	case "number":
		// TODO
	case "integer":
		// TODO
	case "":
		if prop.Ref != "" {
			// TODO
		} else {
			return "", fmt.Errorf("expected blank type to have $ref: %s", propName)
		}
	}

	if retval == "" {
		//return "", errors.New("unable to determine type for " + propName)
		// TODO
		retval = "Todo"
	}
	if !required {
		retval = retval + "?"
	}
	return retval, nil
}

func generatePklClass(typeName string) error {
	schema, err := getSchema(typeName)
	if err != nil {
		return err
	}

	// TODO: Needs to be on a URI somewhere public
	fmt.Println("import cloudformation.pkl")

	classes := make([]*pklDefClass, 0)

	// Iterate over definitions, creating a class for each one
	for name, def := range schema.Definitions {
		cls := &pklDefClass{
			Name:    name,
			Props:   make([]*pklDefProp, 0),
			Aliases: make([]*stringAlias, 0),
		}
		classes = append(classes, cls)

		r := def.GetRequired()

		for propName, prop := range def.Properties {
			required := slices.Contains(r, propName)
			propType, err := getPropType(name, propName, prop, cls, required)
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
	shortName := strings.Split(typeName, "::")[2]
	fmt.Println()
	fmt.Printf("open class %s {\n", shortName)
	fmt.Println()
	fmt.Printf("    Type = %s\n", typeName)
	fmt.Println()

	propNames := make([]string, 0)
	cls := &pklDefClass{
		Name:    shortName,
		Props:   make([]*pklDefProp, 0),
		Aliases: make([]*stringAlias, 0),
	}
	requiredProps := schema.GetRequired()
	for propName, prop := range schema.Properties {
		propPath := "/properties/" + propName
		// Don't emit read-only properties
		if slices.Contains(schema.ReadOnlyProperties, propPath) {
			continue
		}
		required := slices.Contains(requiredProps, propName)
		propType, err := getPropType(shortName, propName, prop, cls, required)
		if err != nil {
			return err
		}
		fmt.Printf("    hidden %s: %s\n", propName, propType)
		propNames = append(propNames, propName)
	}

	fmt.Println()
	fmt.Printf("    Properties {\n")
	for _, propName := range propNames {
		fmt.Printf("        [\"%s\"] = if (%s == null) null else %s\n", propName, propName, propName)
	}
	fmt.Printf("    }\n\n")

	fmt.Println("}")

	return nil
}
