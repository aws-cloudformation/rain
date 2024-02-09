package build

import (
	_ "embed"
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

//go:embed reco/bucket/bucket.yaml
var bucket string

//go:embed reco/pipeline/s3.yaml
var pipelineS3 string

//go:embed reco/pipeline/codecommit.yaml
var pipelineCodeCommit string

var files map[string]string

func output(args []string) {

	files = make(map[string]string)
	files["bucket"] = bucket
	files["pipeline/s3"] = pipelineS3
	files["pipeline/codecommit"] = pipelineCodeCommit

	path := strings.Join(args, "/")

	file, found := files[path]
	if !found {
		fmt.Println(console.Red("Not found"))
		return
	}
	fmt.Println(file)
}

// recommend outputs a recommended template for the chosen use case
func recommend(args []string) {

	// If args are provided, skip the prompt and output the selection
	if len(args) > 0 {
		output(args)
		return
	}

	// TODO: Recursively prompt for selections
	selections := []reco{
		{Name: "bucket", Text: "A secure S3 bucket"},
		{Name: "pipeline", Text: "A CodePipeline pipeline",
			Sub: []reco{
				{Name: "s3", Text: "A pipeline with an S3 source"},
				{Name: "codecommit", Text: "A pipeline with a codecommit source"},
			},
		},
	}

	activeFormat := " {{ .Text | magenta }}"
	selectedFormat := " {{ .Text | blue }}"

	if console.NoColour {
		activeFormat = " {{ .Text }}"
		selectedFormat = " {{ .Text }}"
	}

	prompt := promptui.Select{
		Label: "Select a pattern",
		Items: selections,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   checkIcon + activeFormat,
			Inactive: "   {{ .Text }}",
			Selected: checkIcon + selectedFormat,
		},
	}

	idx, _, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	fmt.Println("You selected: ", selections[idx].Text)
}
