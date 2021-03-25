package pkg

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	cftpkg "github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/spf13/cobra"
)

// Cmd is the merge command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "pkg <template>",
	Short:                 "Package local artifacts into the template through Include:: directives",
	Long:                  "Package local artifacts into the template through Include:: directives",
	Args:                  cobra.ExactArgs(1),
	Annotations:           cmd.TemplateAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fn := args[0]

		template, err := parse.File(fn)
		if err != nil {
			panic(ui.Errorf(err, "unable to open template '%s'", fn))
		}

		spinner.Push(fmt.Sprintf("Packaging template '%s'", fn))
		packaged, err := cftpkg.Template(template)
		if err != nil {
			panic(ui.Errorf(err, "unable to package template '%s'", fn))
		}
		spinner.Pop()

		fmt.Println(format.String(packaged, format.Options{}))
	},
}

func init() {
}
