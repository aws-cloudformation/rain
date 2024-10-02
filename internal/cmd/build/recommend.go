package build

import (
	"embed"
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/manifoldco/promptui"
)

type reco struct {
	Name string
	Text string
	Sub  []reco
}

// We are embedding the entire tmpl directory into the binary as a file system

//go:embed tmpl
var templateFiles embed.FS

func writeFile(args []string) {
	raw := strings.Join(args, "/")
	tmpl := "tmpl/" + raw
	path := tmpl + ".yaml"
	b, err := templateFiles.ReadFile(path)
	if err != nil {
		fmt.Println(console.Red(fmt.Sprintf("Not found: %s", raw)))
	}
	// Package and transform the template to resolve module references
	pkg.Experimental = true
	packaged, err := parse.String(string(b))
	if err != nil {
		fmt.Println(console.Red(err))
		return
	}

	// TODO: Local builds should not make any API calls like zipping
	// files and putting them into an S3 bucket.
	// Either hack the packaging code to not process S3 directives,
	// or make sure none of the samples use them

	transformed, err := pkg.Template(packaged, "tmpl", &templateFiles)
	if err != nil {
		fmt.Println(console.Red(err))
		return
	}
	output(format.CftToYaml(transformed))
}

func showPrompt(selections []reco, path string) {

	p := promptui.Select{
		Label:  "Select a pattern",
		Items:  selections,
		Stdout: NoBellStdout,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   checkIcon + activeFormat,
			Inactive: "  {{ .Name }}: {{ .Text }}",
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
		{Name: "bucket", Text: "S3 buckets",
			Sub: []reco{
				{Name: "bucket", Text: "A secure S3 bucket"},
				{Name: "website", Text: "A static website with a bucket and CloudFront distribution"},
			},
		},
		{Name: "pipeline", Text: "A CodePipeline pipeline",
			Sub: []reco{
				{Name: "s3", Text: "A pipeline with an S3 source"},
				{Name: "codecommit", Text: "A pipeline with a codecommit source"},
			},
		},
		{Name: "ecs", Text: "ECS Clusters",
			Sub: []reco{
				{Name: "cluster", Text: "An ECS Cluster"},
			},
		},
		{Name: "vpc", Text: "Networking",
			Sub: []reco{
				{Name: "vpc", Text: "A VPC with 2 private and 2 public subnets"},
			},
		},
		{Name: "ec2", Text: "EC2 Instances",
			Sub: []reco{
				{Name: "autoscaling", Text: "A VPC, AutoScaling Group, EC2 Launch Template, and ALB for a website"},
			},
		},
		{Name: "eventbridge", Text: "Event Bridge",
			Sub: []reco{
				{Name: "central-logs", Text: "A centralized event bus and log group for collecting logs from org member accounts"},
			},
		},
		{Name: "webapp", Text: "Web Application",
			Sub: []reco{
				{Name: "webapp", Text: "A serverless web application with static content, api gateway, lambda, and dynamodb"},
			},
		},
	}

	showPrompt(selections, "")
}
