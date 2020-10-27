package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/aws-cloudformation/rain/cfn/parse"
	"github.com/spf13/cobra"
)

var compactFlag bool
var jsonFlag bool
var verifyFlag bool
var writeFlag bool
var unsortedFlag bool

var fmtCmd = &cobra.Command{
	Use:                   "fmt <filename>",
	Aliases:               []string{"format"},
	Short:                 "Format CloudFormation templates",
	Long:                  "Reads the named template and outputs a nicely formatted copy.",
	Args:                  cobra.ExactArgs(1),
	Annotations:           templateAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		// Read the template
		fn := args[0]
		input, err := ioutil.ReadFile(fn)
		if err != nil {
			panic(errorf(err, "unable to read '%s'", fn))
		}

		// Parse the template
		source, err := parse.String(string(input))
		if err != nil {
			panic(errorf(err, "unable to parse '%s'", fn))
		}

		// Format the output
		options := format.Options{
			Style:    format.YAML,
			Compact:  compactFlag,
			Unsorted: unsortedFlag,
		}

		if jsonFlag {
			options.Style = format.JSON
		}

		output := format.Template(source, options)

		if verifyFlag {
			if string(input) != output {
				panic(output)
			}

			fmt.Println("Formatted OK")
			return
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
	fmtCmd.Flags().BoolVarP(&compactFlag, "compact", "c", false, "Produce more compact output.")
	fmtCmd.Flags().BoolVarP(&jsonFlag, "json", "j", false, "Output the template as JSON (default format: YAML).")
	fmtCmd.Flags().BoolVarP(&verifyFlag, "verify", "v", false, "Check if the input is already correctly formatted and exit.\nThe exit status will be 0 if so and 1 if not.")
	fmtCmd.Flags().BoolVarP(&writeFlag, "write", "w", false, "Write the output back to the file rather than to stdout.")
	fmtCmd.Flags().BoolVarP(&unsortedFlag, "unsorted", "u", false, "Do not sort the template's properties.")
	Rain.AddCommand(fmtCmd)
}
