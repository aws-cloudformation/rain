package build

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/internal/aws/bedrock"
	"github.com/aws-cloudformation/rain/internal/config"
)

// prompt invokes bedrock to produce a template based on the prompt
func prompt(p string) {
	prompt := fmt.Sprintf("Write an AWS CloudFormation YAML template that builds the following:\n\n%s\n\nDo not include any explanation.\n\nWrite only the content of the YAML file.\n\nOutput valid YAML within <yaml></yaml> tags.", p)
	config.Debugf("About to invoke bedrock claude2 with prompt: %s", prompt)
	r, err := bedrock.Invoke(prompt)
	if err != nil {
		panic(err)
	}

	// Clean up the output
	r = strings.ReplaceAll(r, "<yaml>\n", "")
	r = strings.ReplaceAll(r, "</yaml>", "")
	r = strings.ReplaceAll(r, " AWSTemplateFormatVersion", "AWSTemplateFormatVersion")

	fmt.Println(r)
}
