package cmd

import (
	"strings"

	"github.com/aws-cloudformation/rain/config"
	"github.com/aws-cloudformation/rain/version"
	"github.com/spf13/cobra"
)

// Rain represents the base command when called without any subcommands
var Rain = &cobra.Command{
	Use:     "rain",
	Long:    "Rain is what happens when you have a lot of CloudFormation ;)",
	Version: version.VERSION,
}

const usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

{{range $group := groups}}{{ $group }}:{{range $c := $.Commands}}{{if $c.IsAvailableCommand}}{{if eq $c.Annotations.Group $group}}
  {{rpad $c.Name $c.NamePadding }} {{$c.Short}}{{end}}{{end}}{{end}}

{{end}}Other Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}{{if .Annotations.Group}}{{else}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

func init() {
	Rain.PersistentFlags().StringVarP(&config.Profile, "profile", "p", "", "AWS profile name; read from the AWS CLI configuration file")
	Rain.PersistentFlags().StringVarP(&config.Region, "region", "r", "", "AWS region to use")

	// Customise usage
	Rain.Annotations = map[string]string{"Groups": strings.Join(groups, "|")}

	cobra.AddTemplateFunc("groups", func() []string {
		return groups
	})

	Rain.SetUsageTemplate(usageTemplate)

	addDefaults(Rain)
}
