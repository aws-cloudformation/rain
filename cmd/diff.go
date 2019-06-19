package cmd

import (
	"fmt"

	"github.com/aws-cloudformation/rain/diff"
	"github.com/aws-cloudformation/rain/parse"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:                   "diff <from> <to>",
	Short:                 "Compare CloudFormation templates",
	Long:                  "Outputs a summary of the changes necessary to transform the CloudFormation template named <from> into the template named <to>.",
	Args:                  cobra.ExactArgs(2),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		leftFn, rightFn := args[0], args[1]

		left, err := parse.ReadFile(leftFn)
		if err != nil {
			panic(fmt.Errorf("Unable to parse template '%s': %s", leftFn, err))
		}

		right, err := parse.ReadFile(rightFn)
		if err != nil {
			panic(fmt.Errorf("Unable to parse template '%s': %s", leftFn, err))
		}

		fmt.Print(colouriseDiff(diff.Compare(left, right)))
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
