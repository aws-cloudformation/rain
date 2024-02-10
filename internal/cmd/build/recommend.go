package build

import (
	"embed"
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/manifoldco/promptui"
)

type reco struct {
	Name string
	Text string
	Sub  []reco
}

var checkIcon = "✅"

//go:embed tmpl
var fs embed.FS

func writeFile(args []string) {
	var path string
	raw := strings.Join(args, "/")
	switch raw {
	case "bucket":
		path = "tmpl/bucket/bucket.yaml"
	default:
		path = "tmpl/" + raw + ".yaml"
	}
	b, err := fs.ReadFile(path)
	if err != nil {
		fmt.Println(console.Red(fmt.Sprintf("Not found: %s", raw)))
	}
	output(string(b))
}

func showPrompt(selections []reco, path string) {

	activeFormat := " {{ .Text | magenta }}"
	selectedFormat := " {{ .Text | blue }}"

	if console.NoColour {
		activeFormat = " {{ .Text }}"
		selectedFormat = " {{ .Text }}"
	}

	p := promptui.Select{
		Label: "Select a pattern",
		Items: selections,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   checkIcon + activeFormat,
			Inactive: "   {{ .Text }}",
			Selected: checkIcon + selectedFormat,
		},
	}

	idx, _, err := p.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	selected := selections[idx]

	if path == "" {
		path = selected.Name
	} else {
		path = path + "/" + selected.Name
	}

	if len(selected.Sub) > 0 {
		showPrompt(selected.Sub, path)
	} else {
		writeFile(strings.Split(path, "/"))
	}
}

// recommend outputs a recommended template for the chosen use case
func recommend(args []string) {

	// If args are provided, skip the prompt and output the selection
	if len(args) > 0 {
		writeFile(args)
		return
	}

	// Recursively prompt for selections
	selections := []reco{
		{Name: "bucket", Text: "A secure S3 bucket"},
		{Name: "pipeline", Text: "A CodePipeline pipeline",
			Sub: []reco{
				{Name: "s3", Text: "A pipeline with an S3 source"},
				{Name: "codecommit", Text: "A pipeline with a codecommit source"},
			},
		},
	}

	showPrompt(selections, "")
}
