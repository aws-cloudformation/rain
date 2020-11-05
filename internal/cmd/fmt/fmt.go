package fmt

import (
	"fmt"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/ui"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/spf13/cobra"
)

var jsonFlag bool
var verifyFlag bool
var writeFlag bool
var unsortedFlag bool

// Cmd is the fmt command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "fmt <filename>",
	Aliases:               []string{"format"},
	Short:                 "Format CloudFormation templates",
	Long:                  "Reads the named template and outputs a nicely formatted copy.",
	Args:                  cobra.ExactArgs(1),
	Annotations:           cmd.TemplateAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		// Read the template
		fn := args[0]
		input, err := ioutil.ReadFile(fn)
		if err != nil {
			panic(ui.Errorf(err, "unable to read '%s'", fn))
		}

		// Parse the template
		source, err := parse.String(string(input))
		if err != nil {
			panic(ui.Errorf(err, "unable to parse '%s'", fn))
		}

		// Format the output
		output := format.String(source, format.Options{
			JSON:     jsonFlag,
			Unsorted: unsortedFlag,
		})

		if verifyFlag {
			fmt.Fprint(os.Stderr, fn+": ")

			if strings.TrimSpace(string(input)) == strings.TrimSpace(output) {
				fmt.Fprintln(os.Stderr, "formatted OK")
				os.Exit(0)
			} else {
				fmt.Fprintln(os.Stderr, "would reformat")
				os.Exit(1)
			}
		}

		// Verify the output is valid
		err = parse.Verify(source, output)
		if err != nil {
			panic(err)
		}

		if writeFlag {
			ioutil.WriteFile(fn, []byte(output), 0644)
		} else {
			fmt.Println(output)
		}
	},
}

func init() {
	Cmd.Flags().BoolVarP(&jsonFlag, "json", "j", false, "Output the template as JSON (default format: YAML).")
	Cmd.Flags().BoolVarP(&verifyFlag, "verify", "v", false, "Check if the input is already correctly formatted and exit.\nThe exit status will be 0 if so and 1 if not.")
	Cmd.Flags().BoolVarP(&writeFlag, "write", "w", false, "Write the output back to the file rather than to stdout.")
	Cmd.Flags().BoolVarP(&unsortedFlag, "unsorted", "u", false, "Do not sort the template's properties.")
}
