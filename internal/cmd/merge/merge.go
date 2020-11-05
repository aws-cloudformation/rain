package merge

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/cmd"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/spf13/cobra"
)

var forceMerge = false

// Cmd is the merge command's entrypoint
var Cmd = &cobra.Command{
	Use:                   "merge <template> <template> ...",
	Short:                 "Merge two or more CloudFormation templates",
	Long:                  "Merges all specified CloudFormation templates, print the resultant template to standard out",
	Args:                  cobra.MinimumNArgs(2),
	Annotations:           cmd.TemplateAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		templates := make([]cft.Template, len(args))

		for i, fn := range args {
			templates[i], err = parse.File(fn)
			if err != nil {
				panic(ui.Errorf(err, "unable to open template '%s'", fn))
			}
		}

		var merged cft.Template

		for i, template := range templates {
			if i == 0 {
				merged = template
				continue
			}

			merged, err = mergeTemplates(merged, template)
			if err != nil {
				panic(err)
			}
		}

		fmt.Println(format.String(merged, format.Options{}))
	},
}

func init() {
	Cmd.Flags().BoolVarP(&forceMerge, "force", "f", false, "Don't warn on clashing attributes; rename them instead. Note: this will not rename Refs, GetAtts, etc.")
}
