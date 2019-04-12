package main

import (
	"codecommit/builders/rain/cmd"
	"codecommit/builders/rain/util"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Command func(args ...string)

var plugins map[string]string

var usage string

func init() {
	// Find plugins
	plugins = make(map[string]string)

	path := os.Getenv("PATH")

	for _, dir := range strings.Split(path, ":") {
		bins, err := filepath.Glob(dir + "/cfn-*")
		if err != nil {
			panic(err)
		}

		for _, bin := range bins {
			name := string(bin[len(dir)+5:])

			plugins[name] = bin
		}
	}

	// Prepare usage
	usage = `Usage: rain [COMMAND] [OPTIONS...]

  Rain is a tool to save you some typing when working with CloudFormation
  
  rain is extensible and searches for commands in the following order:
  1. commands built in to the CloudFormation CLI itself
  2. binaries in your path that begin with 'cfn-'
  3. if the command you supply doesn't match 1 or 2, rain runs 'aws cloudformation <command>'

`

	longest := 0

	if len(cmd.Commands) > 0 {
		usage += "Built-in commands:\n\n"

		names := make([]string, 0)

		for name, _ := range cmd.Commands {
			names = append(names, name)

			if len(name) > longest {
				longest = len(name)
			}
		}

		sort.Strings(names)

		for _, name := range names {
			cmd := cmd.Commands[name]
			usage += fmt.Sprintf("  %s  %s- %s\n", name, strings.Repeat(" ", longest-len(name)), cmd.Help)
		}

		usage += "\n"
	}

	if len(plugins) > 0 {
		usage += "Plugins found:\n\n"

		names := make([]string, 0)

		for name, _ := range plugins {
			names = append(names, name)

			if len(name) > longest {
				longest = len(name)
			}
		}

		sort.Strings(names)

		for _, name := range names {
			usage += fmt.Sprintf("  %s  %s- Runs cfn-%s\n", name, strings.Repeat(" ", longest-len(name)), name)
		}

		usage += "\n"
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

	if cmd, ok := cmd.Commands[command]; ok {
		cmd.Func(args)
	} else if plugin, ok := plugins[command]; ok {
		util.RunAttached(plugin, args...)
	} else {
		args = append([]string{"cloudformation", command}, args...)
		util.RunAttached("aws", args...)
	}
}
