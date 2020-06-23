package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/aws-cloudformation/rain/config"
	"github.com/aws-cloudformation/rain/console/spinner"
	"github.com/aws-cloudformation/rain/console/text"
	"github.com/aws-cloudformation/rain/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func addDefaults(c *cobra.Command) {
	// Don't add "[flags]" to the usage line
	c.DisableFlagsInUseLine = true

	// Set version string
	c.Version = version.VERSION

	// Add the debug flag
	c.PersistentFlags().BoolVarP(&config.Debug, "debug", "", false, "Output debugging information")

	// Customise version string
	if c.Name() == "rain" {
		c.SetVersionTemplate(fmt.Sprintf("%s {{.Version}} %s/%s\n",
			version.NAME,
			runtime.GOOS,
			runtime.GOARCH,
		))
	} else {
		c.SetVersionTemplate(fmt.Sprintf("{{.Name}} (%s {{.Version}} %s/%s)\n",
			version.NAME,
			runtime.GOOS,
			runtime.GOARCH,
		))
	}
}

// Wrap creates a new command with the same functionality as src
// but with a new name and default options added for executables
// e.g. the --debug flag
func Wrap(name string, src *cobra.Command) *cobra.Command {
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
	addDefaults(out)

	// Add the flags
	src.Flags().VisitAll(func(f *pflag.Flag) {
		out.Flags().AddFlag(f)
	})

	return out
}

// Execute wraps a command with error trapping that deals with the debug flag
func Execute(cmd *cobra.Command) {
	defer func() {
		spinner.Stop()

		if r := recover(); r != nil {
			if config.Debug {
				panic(r)
			}

			fmt.Println(text.Red(fmt.Sprint(r)))
			os.Exit(1)
		}
	}()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
