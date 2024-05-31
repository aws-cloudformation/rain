package stackset

import (
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/spf13/cobra"
)

const usageTemplate = `Usage:{{if .Runnable}}
  <cyan>{{.UseLine}}</>{{end}}{{if .HasAvailableSubCommands}}
  <cyan>{{.CommandPath}}</> [<gray>command</>]{{end}}{{if gt (len .Aliases) 0}}

Aliases: {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  <cyan>{{rpad .Name .NamePadding }}</> {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
{{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

var delegatedAdmin bool

// addCommand adds a command to the root command.
func addCommand(profileOptions bool, c *cobra.Command) {
	if profileOptions {
		c.Flags().StringVarP(&config.Profile, "profile", "p", "", "AWS profile name; read from the AWS CLI configuration file")
		c.Flags().StringVarP(&config.Region, "region", "r", "", "AWS region to use")
	}

	c.Flags().BoolVar(&delegatedAdmin, "admin", false, "Use delegated admin permissions")

	StackSetCmd.AddCommand(c)
}

var StackSetCmd = &cobra.Command{
	Use:   "stackset <stack_set command>",
	Short: "This command manipulates stack sets.",
	Long:  "This command manipulates stack sets. It has no action if specific stack set command is not added.",
}

func init() {
	addCommand(true, LsCmd)
	addCommand(true, DeployCmd)
	addCommand(true, RmCmd)

	oldUsageFunc := StackSetCmd.UsageFunc()
	StackSetCmd.SetUsageFunc(func(c *cobra.Command) error {
		StackSetCmd.SetUsageTemplate(console.Sprint(usageTemplate))
		return oldUsageFunc(c)
	})

	StackSetCmd.PersistentFlags().BoolVarP(&console.NoColour, "no-colour", "", false, "Disable colour output")
}
