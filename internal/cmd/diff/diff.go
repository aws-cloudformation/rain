package diff

import (
	"fmt"

	"github.com/aws-cloudformation/rain/internal/ui"

	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/spf13/cobra"
)

var longDiff = false

// Cmd is the diff command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "diff <from> <to>",
	Short:                 "Compare CloudFormation templates",
	Long:                  `Outputs a summary of the changes necessary to transform the CloudFormation template named \<from\> into the template named \<to\>.`,
	Args:                  cobra.ExactArgs(2),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		leftFn, rightFn := args[0], args[1]

		left, err := parse.File(leftFn)
		if err != nil {
			panic(ui.Errorf(err, "unable to parse template '%s'", leftFn))
		}

		right, err := parse.File(rightFn)
		if err != nil {
			panic(ui.Errorf(err, "unable to parse template '%s'", leftFn))
		}

		fmt.Print(ui.ColouriseDiff(diff.New(left, right), longDiff))
	},
}

func init() {
	Cmd.Flags().BoolVarP(&longDiff, "long", "l", false, "Include unchanged elements in diff output")
}
