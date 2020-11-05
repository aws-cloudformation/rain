package check

import (
	"errors"
	"fmt"

	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/ui"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/cft/validate"
	"github.com/spf13/cobra"
)

// Cmd is the check command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "check <template file>",
	Short:                 "Validate a CloudFormation template against the spec",
	Long:                  "Reads the specified CloudFormation template and validates it against the current CloudFormation specification.",
	Args:                  cobra.ExactArgs(1),
	Annotations:           cmd.TemplateAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fn := args[0]

		t, err := parse.File(fn)
		if err != nil {
			panic(ui.Errorf(err, "unable to parse template '%s'", fn))
		}

		errs := validate.Template(t)

		if len(errs) == 0 {
			fmt.Printf("%s: ok\n", fn)
		} else {
			for _, err := range errs {
				fmt.Printf("%#v\n", err)
			}

			t.AddComments(errs)
			panic(errors.New(format.String(t, format.Options{})))
		}
	},
}
