package fmt

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/ui"

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
	Long:                  "Reads a CloudFormation template from <filename> (or stdin if no filename is supplied) and formats it",
	Args:                  cobra.MaximumNArgs(1),
	Annotations:           cmd.TemplateAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		var r io.Reader
		var err error

		fn := "<stdin>"

		if len(args) == 0 {
			r = os.Stdin

			// Check there's data
			stat, err := os.Stdin.Stat()
			if err != nil {
				panic(ui.Errorf(err, "unable to open stdin"))
			}

			if stat.Mode()&os.ModeNamedPipe == 0 && stat.Size() == 0 {
				fmt.Println("CAKES")
				panic(cmd.Help())
			}

			writeFlag = false // Can't write back to stdin ;)
		} else {
			fn = args[0]
			r, err = os.Open(args[0])
			if err != nil {
				panic(ui.Errorf(err, "unable to read '%s'", fn))
			}
		}

		// Read the template
		input, err := ioutil.ReadAll(r)
		if err != nil {
			panic(ui.Errorf(err, "unable to read input"))
		}

		// Parse the template
		source, err := parse.String(string(input))
		if err != nil {
			panic(ui.Errorf(err, "unable to parse input"))
		}

		// Format the output
		output := format.String(source, format.Options{
			JSON:     jsonFlag,
			Unsorted: unsortedFlag,
		})

		if verifyFlag {
			if strings.TrimSpace(string(input)) != strings.TrimSpace(output) {
				panic(fmt.Errorf("%s: would reformat", fn))
			}

			fmt.Printf("%s: formatted OK\n", fn)
			return
		}

		// Verify the output is valid
		if err = parse.Verify(source, output); err != nil {
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
