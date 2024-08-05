package build

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/manifoldco/promptui"
)

// promptTypeName uses prompt ui to select from all resource types
func promptTypeName() string {
	selections := make([]buildSelection, 0)

	// Load all type names from a file
	for _, typeName := range strings.Split(cfn.AllTypes, "\n") {
		selections = append(selections, buildSelection{Name: typeName, Text: ""})
	}

	label := "Select the resource type..."
	active := " {{ .Name | magenta }}"
	selected := " {{ .Name | magenta }}"

	if console.NoColour {
		active = " {{ .Name }}"
		selected = " {{ .Name }}"
	}

	// Prompt with a search function so the user can hit / and filter the list
	p := promptui.Select{
		Label:  label,
		Stdout: NoBellStdout,
		Items:  selections,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   checkIcon + active,
			Inactive: "  {{ .Name }}",
			Selected: checkIcon + selected,
		},
		Searcher: func(input string, index int) bool {
			s := selections[index]
			return strings.Contains(strings.ToLower(s.Name), strings.ToLower(input))
		},
	}

	idx, _, err := p.Run()

	if err != nil {
		panic(err)
	}

	return selections[idx].Name
}

// promptPrefix uses prompt ui to ask if the user wants to filter by a prefix
func promptPrefix() string {
	selections := []buildSelection{
		{Name: "Yes", Text: "Yes, let me enter a prefix to filter the list"},
		{Name: "No", Text: "No, list all available types"},
	}

	selection := openPrompt("Do you want to filter the list?", selections)

	if selection == "Yes" {
		prompt := promptui.Prompt{
			Label: "Enter a prefix, such as AWS::S3:",
		}

		result, err := prompt.Run()

		if err != nil {
			panic(err)
		}

		return result

	} else {
		return ""
	}
}

func template() {

	selections := []buildSelection{
		{Name: ALL, Text: "Use all schema properties to output a template with placeholders"},
		{Name: REQUIRED, Text: "Use required schema properties to output a template with placeholders"},
		{Name: PROMPT, Text: "Use Bedrock and Claude to generate a template based on a prompt"},
		{Name: RECOMMEND, Text: "Output a vetted, recommended template for a use case"},
	}

	selected := openPrompt("Select an option to generate the template", selections)

	switch selected {
	case ALL:
		typeName := promptTypeName()
		basicBuild([]string{typeName})
	case REQUIRED:
		bareTemplate = true
		typeName := promptTypeName()
		basicBuild([]string{typeName})
	case PROMPT:
		selections = []buildSelection{
			{Name: CLAUDE2, Text: "Claude 2"},
			{Name: CLAUDE3HAIKU, Text: "Claude 3 Haiku"},
			{Name: CLAUDE3SONNET, Text: "Claude 3 Sonnet"},
			{Name: CLAUDE3OPUS, Text: "Claude 3 Opus"},
			{Name: CLAUDE3_5SONNET, Text: "Claude 3.5 Sonnet"},
		}
		selected = openPrompt("Select a model", selections)
		prompt := promptui.Prompt{
			Label: "Describe the architecture you want to see in the template",
		}
		p, err := prompt.Run()
		if err != nil {
			panic(err)
		}
		model = selected
		runPrompt(p)
	case RECOMMEND:
		recommend([]string{})
	}

}

func policy() {
	selections := []buildSelection{
		{Name: GUARD, Text: "CloudFormation Guard (.guard)"},
		{Name: OPA, Text: "Open Policy Agent (.rego)"},
	}
	lang := openPrompt("Choose a language", selections)
	selections = []buildSelection{
		{Name: CLAUDE3HAIKU, Text: "Claude 3 Haiku"},
		{Name: CLAUDE3SONNET, Text: "Claude 3 Sonnet"},
		{Name: CLAUDE3OPUS, Text: "Claude 3 Opus"},
		{Name: CLAUDE3_5SONNET, Text: "Claude 3.5 Sonnet"},
	}
	model = openPrompt("Choose a model", selections)
	prompt := promptui.Prompt{
		Label: "Describe the policy you want to enforce",
	}
	p, err := prompt.Run()
	if err != nil {
		panic(err)
	}
	switch lang {
	case GUARD:
		promptGuard(p, modelId(model))
	case OPA:
		promptRego(p, modelId(model))
	}
}

func generic() {
	// Choose the model
	selections := []buildSelection{
		{Name: CLAUDE3HAIKU, Text: "Claude 3 Haiku"},
		{Name: CLAUDE3SONNET, Text: "Claude 3 Sonnet"},
		{Name: CLAUDE3OPUS, Text: "Claude 3 Opus"},
		{Name: CLAUDE3_5SONNET, Text: "Claude 3.5 Sonnet"},
	}
	model = openPrompt("Choose a model", selections)

	// Ask if the user wants to include a file
	selections = []buildSelection{
		{Name: YES, Text: "Yes"},
		{Name: NO, Text: "No"},
	}
	yesno := openPrompt("Do you want to include the contents of a file with your question?", selections)

	// Get the file content
	var content string
	if yesno == YES {
		prompt := promptui.Prompt{
			Label: "Enter the file path: ",
		}
		path, err := prompt.Run()
		if err != nil {
			panic(err)
		}
		fmt.Println("Including contents of file:", path)
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		content = string(contentBytes)
	}

	// Get the prompt
	prompt := promptui.Prompt{
		Label: "Prompt: ",
	}
	p, err := prompt.Run()
	if err != nil {
		panic(err)
	}

	if content != "" {
		p += "\n\nReference the file contents below this line.\n\n" + content
	}

	promptGeneric(p, modelId(model))
}

// openPrompt uses promptui to show selections and returns what was selected
func openPrompt(label string, selections []buildSelection) string {

	p := promptui.Select{
		Label:  label,
		Stdout: NoBellStdout,
		Items:  selections,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   checkIcon + activeFormat,
			Inactive: "  {{ .Name }}: {{ .Text }}",
			Selected: checkIcon + selectedFormat,
		},
	}

	idx, _, err := p.Run()

	if err != nil {
		panic(err)
	}

	return selections[idx].Name
}

// interactive shows the user a series of prompts to guide them
// through the process of choosing what they want rain to build
func interactive() {
	/*

		What would you like to do?
		1. Show me a list of all resource types
		2. Output the registry schema for a resource type
		2. Create a CloudFormation template
			a. Use all schema properties to output a template with placeholders
			b. Use required schema properties to output a template with placeholders
			c. Use Bedrock and Claude to generate a template based on a propmt
				i. Claude 2
				ii. Claude 3 Haiku
				iii. Claude 3 Sonnet
				iv. Claude 3 Opus
				iv. Claude 3.5 Sonnet
			d. Output a vetted recommended template for a use case
		3. Create a policy validation file
		    a. Guard
			b. OPA
				i. (Sonnet or Haiku or Opus, not Claude2)

	*/

	selections := []buildSelection{
		{Name: LIST, Text: "Show me a list of all resource types"},
		{Name: SCHEMA, Text: "Output the schema for a resource type"},
		{Name: TEMPLATE, Text: "Create a CloudFormation template"},
		{Name: POLICY, Text: "Create a policy validation file"},
		{Name: GENERIC, Text: "Ask a bedrock model a generic question"},
	}

	label := "Entering build interactive mode... what would you like to do?"
	selected := openPrompt(label, selections)

	switch selected {
	case LIST:
		list(promptPrefix())
		return
	case SCHEMA:
		schema(promptTypeName())
		return
	case TEMPLATE:
		template()
		return
	case POLICY:
		policy()
		return
	case GENERIC:
		generic()
		return
	}

}

type buildSelection struct {
	Name string
	Text string
}

const (
	LIST               = "list"
	SCHEMA             = "schema"
	TEMPLATE           = "template"
	POLICY             = "policy"
	ALL                = "all"
	REQUIRED           = "required"
	RECOMMEND          = "recommend"
	PROMPT             = "prompt"
	CLAUDE2            = "claude2"
	CLAUDE3OPUS        = "claude3opus"
	CLAUDE3SONNET      = "claude3sonnet"
	CLAUDE3HAIKU       = "claude3haiku"
	CLAUDE3_5SONNET    = "claude3.5sonnet"
	GUARD              = "guard"
	OPA                = "opa"
	GENERIC            = "generic"
	YES                = "yes"
	NO                 = "no"
)
