package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/util"
)

func init() {
	// Find plugins
	path := os.Getenv("PATH")

	for _, dir := range strings.Split(path, ":") {
		bins, err := filepath.Glob(dir + "/cfn-*")
		if err != nil {
			panic(err)
		}

		for _, bin := range bins {
			name := string(bin[len(dir)+5:])

			Commands[name] = Command{
				Type: PLUGIN,
				Help: fmt.Sprintf("Runs %s", bin[len(dir)+1:]),
				Run: func(args []string) {
					util.RunAttached(bin, args...)
				},
			}
		}
	}
}
