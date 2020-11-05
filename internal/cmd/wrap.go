package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// AddDefaults add standard additional flags and version information to a command
func AddDefaults(c *cobra.Command) {
	// Don't add "[flags]" to the usage line
	c.DisableFlagsInUseLine = true

	// Set version string
	c.Version = config.VERSION

	// Add the debug flag
	c.PersistentFlags().BoolVarP(&config.Debug, "debug", "", false, "Output debugging information")

	// Add the no colour flag
	c.PersistentFlags().BoolVarP(&console.NoColour, "no-colour", "", false, "Disable colour output")

	// Customise version string
	if c.Name() == "rain" {
		c.SetVersionTemplate(fmt.Sprintf("%s {{.Version}} %s/%s\n",
			config.NAME,
			runtime.GOOS,
			runtime.GOARCH,
		))
	} else {
		c.SetVersionTemplate(fmt.Sprintf("{{.Name}} (%s {{.Version}} %s/%s)\n",
			config.NAME,
			runtime.GOOS,
			runtime.GOARCH,
		))
	}
}

// Wrap creates a new command with the same functionality as src
// but with a new name and default options added for executables
// e.g. the --debug flag
// The new command is then executed
func Wrap(name string, src *cobra.Command) {
	use := strings.Split(src.Use, " ")
	use[0] = name

	// Create the new command
	out := &cobra.Command{
		Use:  strings.Join(use, " "),
		Long: src.Long,
		Args: src.Args,
		Run:  src.Run,
	}

	// Set default options
	AddDefaults(out)

	// Add the flags
	src.Flags().VisitAll(func(f *pflag.Flag) {
		out.Flags().AddFlag(f)
	})

	Execute(out)
}

// Execute wraps a command with error trapping that deals with the debug flag
func Execute(cmd *cobra.Command) {
	defer func() {
		spinner.Stop()

		if r := recover(); r != nil {
			if config.Debug {
				panic(r)
			}

			fmt.Fprintln(os.Stderr, console.Red(fmt.Sprintf("%s", r)))
			os.Exit(1)
		}
	}()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
