package fmt

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/ui"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/spf13/cobra"
)

var jsonFlag bool
var verifyFlag bool
var writeFlag bool
var unsortedFlag bool

type result struct {
	name   string
	output string
	ok     bool
	err    error
}

func formatReader(name string, r io.Reader) result {
	res := result{
		name: name,
	}

	// Read the template
	input, err := ioutil.ReadAll(r)
	if err != nil {
		res.err = ui.Errorf(err, "unable to read input")
		return res
	}

	// Parse the template
	source, err := parse.String(string(input))
	if err != nil {
		res.err = ui.Errorf(err, "unable to parse input")
		return res
	}

	// Format the output
	res.output = format.String(source, format.Options{
		JSON:     jsonFlag,
		Unsorted: unsortedFlag,
	})

	// Verify the output is valid
	if err = parse.Verify(source, res.output); err != nil {
		res.err = err
		return res
	}

	res.ok = strings.TrimSpace(string(input)) == strings.TrimSpace(res.output)

	return res
}

func formatFile(filename string) result {
	r, err := os.Open(filename)
	if err != nil {
		return result{
			name: filename,
			err:  ui.Errorf(err, "unable to read '%s'", filename),
		}
	}

	return formatReader(filename, r)
}

// Cmd is the fmt command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "fmt <filename>...",
	Aliases:               []string{"format"},
	Short:                 "Format CloudFormation templates",
	Long:                  "Reads CloudFormation templates from filename arguments (or stdin if no filenames are supplied) and formats them",
	Annotations:           cmd.TemplateAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		var results []result

		if len(args) == 0 {
			// Check there's data on stdin
			stat, err := os.Stdin.Stat()
			if err != nil {
				panic(ui.Errorf(err, "unable to open stdin"))
			}

			if stat.Mode()&os.ModeNamedPipe == 0 && stat.Size() == 0 {
				fmt.Println("CAKES")
				panic(cmd.Help())
			}

			writeFlag = false // Can't write back to stdin ;)

			results = []result{
				formatReader("<stdin>", os.Stdin),
			}
		} else {
			results = make([]result, len(args))
			for i, filename := range args {
				results[i] = formatFile(filename)
			}
		}

		hasErr := false

		for i, res := range results {
			if res.err != nil {
				fmt.Fprintln(os.Stderr, console.Red(res.err))
				hasErr = true
				break
			}

			if verifyFlag {
				if res.ok {
					fmt.Println(console.Green(fmt.Sprintf("%s: formatted OK\n", res.name)))
				} else {
					fmt.Fprintln(os.Stderr, console.Red(fmt.Sprintf("%s: would reformat\n", res.name)))
					hasErr = true
				}
			} else if writeFlag {
				ioutil.WriteFile(res.name, []byte(res.output), 0644)
			} else {
				if len(args) > 1 {
					fmt.Printf("--- # %s\n", res.name)
				}

				fmt.Print(res.output)

				if len(args) > 1 && i == len(args)-1 {
					fmt.Println("...")
				}
			}
		}

		if hasErr {
			os.Exit(1)
		}
	},
}

func init() {
	Cmd.Flags().BoolVarP(&jsonFlag, "json", "j", false, "Output the template as JSON (default format: YAML).")
	Cmd.Flags().BoolVarP(&verifyFlag, "verify", "v", false, "Check if the input is already correctly formatted and exit.\nThe exit status will be 0 if so and 1 if not.")
	Cmd.Flags().BoolVarP(&writeFlag, "write", "w", false, "Write the output back to the file rather than to stdout.")
	Cmd.Flags().BoolVarP(&unsortedFlag, "unsorted", "u", false, "Do not sort the template's properties.")
}
