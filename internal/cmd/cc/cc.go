package cc

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/dc"
	"github.com/spf13/cobra"
)

// Args
var params []string
var tags []string
var configFilePath string
var Experimental bool
var yes bool
var ignoreUnknownParams bool
var downloadState bool
var unlock string

// Globals (seems bad..? but cumbersome to pass them around)
var deployedTemplate cft.Template
var resMap map[string]*Resource
var templateConfig *dc.DeployConfig

var Cmd = &cobra.Command{
	Use:   "cc <command>",
	Short: "Interact with templates using Cloud Control API instead of CloudFormation",
	Long: `You must pass the --experimental (-x) flag to use this command, to acknowledge that it is experimental and likely to be unstable!
`,
	Run: run,
}

func run(cmd *cobra.Command, args []string) {
	fmt.Println("Usage: cc deploy|rm|state")
}

func init() {
	Cmd.AddCommand(CCDeployCmd)
	Cmd.AddCommand(CCRmCmd)
	Cmd.AddCommand(CCStateCmd)
}
