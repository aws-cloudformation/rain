package stackset

import (
	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/spf13/cobra"
)

const usageTemplate = `Usage:{{if .Runnable}}
  <cyan>{{.UseLine}}</>{{end}}{{if .HasAvailableSubCommands}}
  <cyan>{{.CommandPath}}</> [<gray>command</>]{{end}}{{if gt (len .Aliases) 0}}

Aliases: 
{{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples: {{.Example}}{{end}}
{{if .HasAvailableSubCommands}} 
Available commands:
  {{range $c := $.Commands}}{{if $c.IsAvailableCommand}}<cyan>{{rpad $c.Name $c.NamePadding }}</> {{$c.Short}}{{end}}{{end}}

Flags:
{{.Flags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}

{{end}}
`

func addCommand(profileOptions bool, c *cobra.Command) {
	if profileOptions {
		c.Flags().StringVarP(&config.Profile, "profile", "p", "", "AWS profile name; read from the AWS CLI configuration file")
		c.Flags().StringVarP(&config.Region, "region", "r", "", "AWS region to use")
	}

	StackSetCmd.AddCommand(c)
}

var StackSetCmd = &cobra.Command{
	Use:   "stackset <stack_set command>",
	Short: "This command allows to manipulate with stack sets.",
	Long:  "This command allows to manipulate with stack sets. It has no action if specific stack set command is not added.",
}

func init() {
	addCommand(true, LsCmd)

	oldUsageFunc := StackSetCmd.UsageFunc()
	StackSetCmd.SetUsageFunc(func(c *cobra.Command) error {
		StackSetCmd.SetUsageTemplate(console.Sprint(usageTemplate))
		return oldUsageFunc(c)
	})

	StackSetCmd.PersistentFlags().BoolVarP(&console.NoColour, "no-colour", "", false, "Disable colour output")

	cmd.AddDefaults(StackSetCmd)
}
