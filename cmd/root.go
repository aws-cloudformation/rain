package cmd

import (
	"os"
	"strings"

	"github.com/aws-cloudformation/rain/config"
	"github.com/spf13/cobra"
)

const (
	stackGroup    = "Stack commands"
	templateGroup = "Template commands"
)

var groups = []string{
	stackGroup,
	templateGroup,
}

type Command struct {
	cobra.Command
}

// Root represents the base command when called without any subcommands
var Root = &cobra.Command{
	Use:     "rain",
	Long:    "Rain is a development workflow tool for working with AWS CloudFormation.",
	Version: "v0.7.2",
}

const rootUsageTemplate = `Usage: {{.UseLine}} [command]
{{range $group := groups}}
{{ $group }}: {{range $c := $.Commands}}{{if $c.IsAvailableCommand}}{{if eq $c.Annotations.Group $group}}
  {{rpad $c.Name $c.NamePadding }} {{$c.Short}}{{end}}{{end}}{{end}}
{{end}}
Other Commands: {{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}{{if .Annotations.Group}}{{else}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
`

func init() {
	Root.PersistentFlags().BoolVarP(&config.Debug, "debug", "", false, "Output debugging information")
	Root.PersistentFlags().StringVarP(&config.Profile, "profile", "p", "", "AWS profile name; read from the AWS CLI configuration file")
	Root.PersistentFlags().StringVarP(&config.Region, "region", "r", "", "AWS region to use")

	Root.Annotations = map[string]string{"Groups": strings.Join(groups, "|")}

	cobra.AddTemplateFunc("groups", func() []string {
		return groups
	})

	Root.SetUsageTemplate(rootUsageTemplate)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the Root.
func Execute() {
	if err := Root.Execute(); err != nil {
		os.Exit(1)
	}
}
