package pkg

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft/format"
	cftpkg "github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/spf13/cobra"
)

// Cmd is the merge command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "pkg <template>",
	Short:                 "Package local artifacts into the template through Rain:: directives",
	Long:                  "Package local artifacts into the template through Rain:: directives",
	Args:                  cobra.ExactArgs(1),
	Annotations:           cmd.TemplateAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fn := args[0]

		spinner.Push(fmt.Sprintf("Packaging template '%s'", fn))
		packaged, err := cftpkg.File(fn)
		if err != nil {
			panic(ui.Errorf(err, "unable to package template '%s'", fn))
		}
		spinner.Pop()

		fmt.Println(format.String(packaged, format.Options{}))
	},
}

func init() {
}
