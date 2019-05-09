package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/aws-cloudformation/rain/cmd"
	"github.com/aws-cloudformation/rain/util"
)

var usage string

func init() {
	// Prepare usage
	usage = `Usage: rain [COMMAND] [OPTIONS...]

  Rain is a tool to save you some typing when working with CloudFormation
  
  rain is extensible and searches for commands in the following order:
  1. commands built in to the CloudFormation CLI itself
  2. binaries in your path that begin with 'cfn-'
  3. if the command you supply doesn't match 1 or 2, rain runs 'aws cloudformation <command>'

`

	for _, cmdType := range cmd.CommandTypes {
		names := make([]string, 0)
		longest := 0

		for name, cmd := range cmd.Commands {
			if cmd.Type == cmdType {
				names = append(names, name)

				if len(name) > longest {
					longest = len(name)
				}
			}
		}

		if len(names) > 0 {
			sort.Strings(names)

			usage += fmt.Sprintf("%s commands:\n", cmdType)

			for _, name := range names {
				cmd := cmd.Commands[name]
				usage += fmt.Sprintf("  %s  %s- %s\n", name, strings.Repeat(" ", longest-len(name)), cmd.Help)
			}

			usage += "\n"
		}
	}
}

func die() {
	fmt.Fprint(os.Stderr, usage)
	os.Exit(1)
}

func main() {
	args := os.Args[1:]

	if len(args) < 1 {
		die()
	}

	command := args[0]
	args = args[1:]

	if c, ok := cmd.Commands[command]; ok {
		c.Run(args)
	} else {
		args = append([]string{"cloudformation", command}, args...)
		util.RunAttached("aws", args...)
	}
}
