package fmt

import (
	"fmt"
	"io"
	"os"
	"strings"

	rainpkl "github.com/aws-cloudformation/rain/pkl"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/ui"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/spf13/cobra"
)

var jsonFlag bool
var pklFlag bool
var verifyFlag bool
var writeFlag bool
var unsortedFlag bool
var dataModel bool
var uncdk bool

// pklPackageAlias is the package name to use in module imports
var pklPackageAlias string = "@cfn"

// pklBasic is set by the --pkl-basic CLI arg to emit Pkl without any imports
var pklBasic bool = false

type result struct {
	name   string
	output string
	ok     bool
	err    error
}

func formatString(input string, res *result) {

	// Parse the template
	source, err := parse.String(string(input))
	if err != nil {
		res.err = ui.Errorf(err, "unable to parse input")
		return
	}

	config.Debugf("%s", node.ToSJson(source.Node))

	if uncdk {
		// Remove CDK Metadata
		// Simplify logical IDs
		err := format.UnCDK(source)
		if err != nil {
			res.err = ui.Errorf(err, "uncdk failed")
			return
		}
	}

	if dataModel {
		res.output = node.ToJson(source.Node)
	} else if pklFlag {
		res.output, err = format.CftToPkl(source, pklBasic, pklPackageAlias)
		if err != nil {
			res.err = err
			return
		}
	} else {
		// Format the output
		res.output = format.String(source, format.Options{
			JSON:     jsonFlag,
			Unsorted: unsortedFlag,
		})

		// Verify the output is valid
		if err = parse.Verify(source, res.output); err != nil {
			res.err = err
			return
		}

		res.ok = strings.TrimSpace(string(input)) == strings.TrimSpace(res.output)
	}
}

func formatReader(name string, r io.Reader) result {
	res := result{
		name: name,
	}

	// Read the template
	input, err := io.ReadAll(r)
	if err != nil {
		res.err = ui.Errorf(err, "unable to read input")
		return res
	}

	formatString(string(input), &res)

	return res
}

func formatFile(filename string) result {

	if strings.HasSuffix(filename, ".pkl") {

		res := result{
			name: filename,
		}

		yaml, err := rainpkl.Yaml(filename)
		if err != nil {
			panic(err)
		}

		formatString(yaml, &res)

		return res
	}

	var r *os.File
	var err error

	if filename == "-" {
		r = os.Stdin
		filename = "stdin"
	} else {
		r, err = os.Open(filename)
		if err != nil {
			return result{
				name: filename,
				err:  ui.Errorf(err, "unable to read '%s'", filename),
			}
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
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		var results []result

		if len(args) == 0 {
			// Check there's data on stdin
			stat, err := os.Stdin.Stat()
			if err != nil {
				cmd.Help()
				os.Exit(0)
			}

			if stat.Mode()&os.ModeNamedPipe == 0 && stat.Size() == 0 {
				cmd.Help()
				os.Exit(0)
			}

			// There is data on stdin, assume the user intends to format it
			args = []string{"-"}
		}

		results = make([]result, len(args))
		for i, filename := range args {
			results[i] = formatFile(filename)
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
					fmt.Println(console.Green(fmt.Sprintf("%s: formatted OK", res.name)))
				} else {
					fmt.Fprintln(os.Stderr, console.Red(fmt.Sprintf("%s: would reformat", res.name)))
					hasErr = true
				}
			} else if writeFlag {
				os.WriteFile(res.name, []byte(res.output), 0644)
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
	Cmd.Flags().BoolVarP(&pklFlag, "pkl", "p", false, "Output the template as Pkl (default format: YAML).")
	Cmd.Flags().BoolVar(&pklBasic, "pkl-basic", false, "Don't use Pkl modules for output")
	Cmd.Flags().BoolVarP(&verifyFlag, "verify", "v", false, "Check if the input is already correctly formatted and exit.\nThe exit status will be 0 if so and 1 if not.")
	Cmd.Flags().BoolVarP(&writeFlag, "write", "w", false, "Write the output back to the file rather than to stdout.")
	Cmd.Flags().BoolVarP(&unsortedFlag, "unsorted", "u", false, "Do not sort the template's properties.")
	Cmd.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
	Cmd.Flags().BoolVar(&dataModel, "datamodel", false, "Output the go yaml data model")
	Cmd.Flags().StringVar(&pklPackageAlias, "pkl-package", "@cfn", "An alias or full package URI for the Pkl package for generated Pkl files")
	Cmd.Flags().StringVar(&format.NodeStyle, "node-style", "", format.NodeStyleDocs)
	Cmd.Flags().BoolVar(&uncdk, "uncdk", false, "Remove CDK Metadata and simplify logical ids")
}
