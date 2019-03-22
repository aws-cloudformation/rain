package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Command func(args ...string)

var plugins map[string]string

var commands = map[string]Command{}

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
	usage = `Usage: cfn [COMMAND] [OPTIONS...]

  The CloudFormation CLI is a tool to save you some typing when working with CloudFormation
  
  cfn is extensible and searches for commands in the following order:
  1. commands built in to the CloudFormation CLI itself
  2. binaries in your path that begin with 'cfn-'
  3. if the command you supply doesn't match 1 or 2, cfn runs 'aws cloudformation <command>'

`

	longest := 0

	if len(commands) > 0 {
		usage += "Built-in commands:\n\n"

		for name, _ := range commands {
			if len(name) > longest {
				longest = len(name)
			}
		}

		for name, _ := range commands {
			usage += fmt.Sprintf("  %s  %s- %s\n", name, strings.Repeat(" ", longest-len(name)), name)
		}

		usage += "\n"
	}

	if len(plugins) > 0 {
		usage += "Plugins found:\n\n"

		for name, _ := range plugins {
			if len(name) > longest {
				longest = len(name)
			}
		}

		for name, _ := range plugins {
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

	if cmdFunc, ok := commands[command]; ok {
		cmdFunc(args[1:]...)
	} else if plugin, ok := plugins[command]; ok {
		fmt.Println("Execing:", plugin, args[1:])
	} else {
		die()
	}
}
