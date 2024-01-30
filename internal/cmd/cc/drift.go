package cc

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/spf13/cobra"
)

func runDrift(cmd *cobra.Command, args []string) {

	name := args[0]

	if !Experimental {
		panic("Please add the --experimental arg to use this feature")
	}

	bucketName := s3.RainBucket(true)

	key := fmt.Sprintf("%v/%v.yaml", STATE_DIR, name) // deployments/name

	obj, err := s3.GetObject(bucketName, key)
	if err != nil {
		fmt.Printf("Unable to download state: %v", err)
		return
	}

	config.Debugf("State file: %s", obj)

	template, err := parse.String(string(obj))
	if err != nil {
		panic(err)
	}

	_, resources := s11n.GetMapValue(template.Node.Content[0], "Resources")
	if resources == nil {
		panic("unable to locate the Resources node")
	}

	for k, v := range resources.Content {
		fmt.Printf("%v: %v", k, v)
	}

}

var CCDriftCmd = &cobra.Command{
	Use:   "drift <name>",
	Short: "Compare the state file to the live state of the resources",
	Long: `When deploying templates with the cc command, a state file is created and stored in the rain assets bucket. This command outputs a diff of that file and the actual state of the resources, according to Cloud Control API.
`,
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run:                   runDrift,
}

func init() {
	CCDriftCmd.Flags().BoolVar(&config.Debug, "debug", false, "Output debugging information")
	CCDriftCmd.Flags().BoolVarP(&Experimental, "experimental", "x", false, "Acknowledge that this is an experimental feature")
}
