package pkg

import (
	"fmt"
	"os"

	"github.com/aws-cloudformation/rain/cft/format"
	cftpkg "github.com/aws-cloudformation/rain/cft/pkg"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/ui"
	"github.com/spf13/cobra"
)

var outFn = ""

// Cmd is the pkg command's entrypoint
var Cmd = &cobra.Command{
	Use:   "pkg <template>",
	Short: "Package local artifacts into a template",
	Long: `Performs the same functions as "aws cloudformation package" but with added functionality.

You may use the following, rain-specific directives in templates packaged with "rain pkg":

  !Rain::Embed <path>          Embeds the contents of the file at <path> into the template as a string

  !Rain::Include <path>        Reads the file at <path> as YAML/JSON and inserts the resulting object into the template

  !Rain::S3Http <path>         Uploads <path> (zipping first if it is a directory) to S3
                               and embeds the S3 HTTP URL into the template as a string

  !Rain::S3 <path>             Uploads <path> (zipping first if it is a directory) to S3
                               and embeds the S3 URI into the template as a string

  !Rain::S3 <object>           supply an object with the following properties: 
    Path: <path>               a file or directory to be uploaded to S3
    Zip: true|false            If "true", rain with zip <path> even if it is a file
    BucketProperty: <bucket>   If you supply "BucketProperty" and "KeyProperty", rain pkg will
    KeyProperty: <key>         include the uploaded file/directory's details as an object in the template
                               with the property names you specify.
    Format: Uri|Http           Specify which format rain pkg should return the S3 location as.
                               Do not specify this property if you supply BucketProperty and KeyProperty.
                               The default Format is "Uri".

  !Rain::Module <url>		   Supply a URL to a rain module, which is similar to a CloudFormation module, 
  						       but allows for type inheritance. One of the resources in the module yaml file 
							   must be called "ModuleExtension", and it must have a Metadata entry called 
							   "Extends" that supplies the existing type to be extended. The Parameters section 
							   of the module can be used to define additional properties for the extension.
`,
	Args:                  cobra.ExactArgs(1),
	Aliases:               []string{"package"},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fn := args[0]

		spinner.Push(fmt.Sprintf("Packaging template '%s'", fn))
		packaged, err := cftpkg.File(fn)
		if err != nil {
			panic(ui.Errorf(err, "unable to package template '%s'", fn))
		}
		spinner.Pop()

		out := format.String(packaged, format.Options{})

		if outFn != "" {
			os.WriteFile(outFn, []byte(out), 0644)
		} else {
			fmt.Println(out)
		}
	},
}

func init() {
	Cmd.Flags().StringVarP(&outFn, "output", "o", "", "Output packaged template to a file")
	Cmd.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
}
